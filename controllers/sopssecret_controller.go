/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/snapp-incubator/sops-operator/lang"
	"io/ioutil"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.mozilla.org/sops/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gitopssecretsnappcloudiov1alpha1 "github.com/snapp-incubator/sops-operator/api/v1alpha1"
	sopsaes "go.mozilla.org/sops/v3/aes"
	sopslogging "go.mozilla.org/sops/v3/logging"
	sopsdotenv "go.mozilla.org/sops/v3/stores/dotenv"
	sopsjson "go.mozilla.org/sops/v3/stores/json"
	sopsyaml "go.mozilla.org/sops/v3/stores/yaml"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SopsSecretReconciler reconciles a SopsSecret object
type SopsSecretReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Log          logr.Logger
	RequeueAfter int64
}

//+kubebuilder:rbac:groups=gitopssecret.snappcloud.io,resources=sopssecrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gitopssecret.snappcloud.io,resources=sopssecrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gitopssecret.snappcloud.io,resources=sopssecrets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SopsSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *SopsSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("sopssecret", req.NamespacedName)

	r.Log.Info("Reconciling", "sopssecret", req.NamespacedName)

	encryptedSopsSecret, finishReconcileLoop, err := r.getEncryptedSopsSecret(ctx, req)
	if finishReconcileLoop {
		return reconcile.Result{}, err
	}

	referencedGPGKey, rescheduleReconcileLoop := r.getGPGKeyRefNameObj(ctx, req, encryptedSopsSecret)
	if rescheduleReconcileLoop {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	if r.isSecretSuspended(encryptedSopsSecret, req) {
		return reconcile.Result{}, nil
	}

	plainTextSopsSecret, rescheduleReconcileLoop := r.decryptSopsSecret(encryptedSopsSecret, referencedGPGKey.Spec.Passphrase)
	if rescheduleReconcileLoop {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	// Iterate over secret templates
	r.Log.Info("Entering template data loop", "sopssecret", req.NamespacedName)
	stringData := plainTextSopsSecret.Spec.StringData

	kubeSecretFromTemplate, rescheduleReconcileLoop := r.newKubeSecretFromTemplate(req, encryptedSopsSecret, plainTextSopsSecret, &stringData)
	if rescheduleReconcileLoop {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	kubeSecretInCluster, rescheduleReconcileLoop := r.getSecretFromClusterOrCreateFromTemplate(ctx, req, encryptedSopsSecret, kubeSecretFromTemplate)
	if rescheduleReconcileLoop {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	rescheduleReconcileLoop = r.isKubeSecretManagedOrAnnotatedToBeManaged(req, encryptedSopsSecret, kubeSecretInCluster)
	if rescheduleReconcileLoop {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	rescheduleReconcileLoop = r.refreshKubeSecretIfNeeded(ctx, req, encryptedSopsSecret, kubeSecretFromTemplate, kubeSecretInCluster)
	if rescheduleReconcileLoop {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}

	encryptedSopsSecret.Status.Message = "Healthy"
	_ = r.Status().Update(context.Background(), encryptedSopsSecret)

	r.Log.Info("SopsSecret is Healthy", "sopssecret", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *SopsSecretReconciler) getGPGKeyRefNameObj(
	ctx context.Context,
	req ctrl.Request,
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
) (*gitopssecretsnappcloudiov1alpha1.GPGKey, bool) {
	gpgkey := &gitopssecretsnappcloudiov1alpha1.GPGKey{}
	namespacedName := types.NamespacedName{Namespace: req.Namespace, Name: encryptedSopsSecret.Spec.GPGKeyRefName}
	err := r.Get(ctx, namespacedName, gpgkey)
	if err != nil {
		r.Log.Info("Error fetching GPGKey", "GPGKey", namespacedName, "error", err)
		encryptedSopsSecret.Status.Message = lang.ErrGPGKeyRefFetchFail
		_ = r.Status().Update(ctx, encryptedSopsSecret)
		return nil, true
	}
	return gpgkey, false
}

func (r *SopsSecretReconciler) decryptSopsSecret(
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	passphrase string,
) (*gitopssecretsnappcloudiov1alpha1.SopsSecret, bool) {
	decryptedSopsSecret, err := decryptSopsSecretInstance(encryptedSopsSecret, r.Log, passphrase)
	if err != nil {
		encryptedSopsSecret.Status.Message = "Decryption error"

		// will not process plainTextSopsSecret error as we are already in error mode here
		_ = r.Status().Update(context.Background(), encryptedSopsSecret)

		// Failed to decrypt, re-schedule reconciliation in 5 minutes
		return nil, true
	}
	return decryptedSopsSecret, false
}

func (r *SopsSecretReconciler) isKubeSecretManagedOrAnnotatedToBeManaged(
	req ctrl.Request,
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	kubeSecretInCluster *corev1.Secret,
) bool {
	// kubeSecretFromTemplate found - perform ownership check
	if !metav1.IsControlledBy(kubeSecretInCluster, encryptedSopsSecret) && !isAnnotatedToBeManaged(kubeSecretInCluster) {
		encryptedSopsSecret.Status.Message = "Child secret is not owned by controller error"
		r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.Info(
			"Child secret is not owned by controller or sopssecret Error",
			"sopssecret", req.NamespacedName,
			"error", fmt.Errorf("sopssecret has a conflict with existing kubernetes secret resource, potential reasons: target secret already pre-existed or is managed by multiple sops secrets"),
		)
		return true
	}
	return false
}

func (r *SopsSecretReconciler) refreshKubeSecretIfNeeded(
	ctx context.Context,
	req ctrl.Request,
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	kubeSecretFromTemplate *corev1.Secret,
	kubeSecretInCluster *corev1.Secret,
) bool {
	copyOfKubeSecretInCluster := kubeSecretInCluster.DeepCopy()

	copyOfKubeSecretInCluster.StringData = kubeSecretFromTemplate.StringData
	copyOfKubeSecretInCluster.Data = map[string][]byte{}
	copyOfKubeSecretInCluster.Type = kubeSecretFromTemplate.Type
	copyOfKubeSecretInCluster.ObjectMeta.Annotations = kubeSecretFromTemplate.ObjectMeta.Annotations
	copyOfKubeSecretInCluster.ObjectMeta.Labels = kubeSecretFromTemplate.ObjectMeta.Labels

	if isAnnotatedToBeManaged(kubeSecretInCluster) {
		copyOfKubeSecretInCluster.ObjectMeta.OwnerReferences = kubeSecretFromTemplate.ObjectMeta.OwnerReferences
	}

	if !apiequality.Semantic.DeepEqual(kubeSecretInCluster, copyOfKubeSecretInCluster) {
		r.Log.Info(
			"Secret already exists and needs to be refreshed",
			"secret", copyOfKubeSecretInCluster.Name,
			"namespace", copyOfKubeSecretInCluster.Namespace,
		)
		if err := r.Update(ctx, copyOfKubeSecretInCluster); err != nil {
			encryptedSopsSecret.Status.Message = "Child secret update error"
			r.Status().Update(context.Background(), encryptedSopsSecret)

			r.Log.Info(
				"Child secret update error",
				"sopssecret", req.NamespacedName,
				"error", err,
			)
			return true
		}
		r.Log.Info(
			"Secret successfully refreshed",
			"secret", copyOfKubeSecretInCluster.Name,
			"namespace", copyOfKubeSecretInCluster.Namespace,
		)
	}
	return false
}

func (r *SopsSecretReconciler) getSecretFromClusterOrCreateFromTemplate(
	ctx context.Context,
	req ctrl.Request,
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	kubeSecretFromTemplate *corev1.Secret,
) (*corev1.Secret, bool) {

	// Check if kubeSecretFromTemplate already exists in the cluster store
	kubeSecretToFindAndCompare := &corev1.Secret{}
	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      kubeSecretFromTemplate.Name,
			Namespace: kubeSecretFromTemplate.Namespace,
		},
		kubeSecretToFindAndCompare,
	)

	// No kubeSecretFromTemplate alike found - CREATE one
	if errors.IsNotFound(err) {
		r.Log.Info(
			"Creating a new Secret",
			"sopssecret", req.NamespacedName,
			"message", err,
		)
		err = r.Create(ctx, kubeSecretFromTemplate)
		kubeSecretToFindAndCompare = kubeSecretFromTemplate.DeepCopy()
	}

	// Unknown error while trying to find kubeSecretFromTemplate in cluster - reschedule reconciliation
	if err != nil {
		encryptedSopsSecret.Status.Message = "Unknown Error"
		r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.Info(
			"Unknown Error",
			"sopssecret", req.NamespacedName,
			"error", err,
		)
		return nil, true
	}

	return kubeSecretToFindAndCompare, false
}

func (r *SopsSecretReconciler) newKubeSecretFromTemplate(
	req ctrl.Request,
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	plainTextSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	stringData *map[string]string,
) (*corev1.Secret, bool) {

	// Define a new secret object
	kubeSecretFromTemplate, err := createKubeSecretFromTemplate(plainTextSopsSecret, stringData, r.Log)
	if err != nil {
		encryptedSopsSecret.Status.Message = "New child secret creation error"
		r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.Info(
			"New child secret creation error",
			"sopssecret", req.NamespacedName,
			"error", err,
		)
		return nil, true
	}

	// Set encryptedSopsSecret as the owner of kubeSecret
	err = controllerutil.SetControllerReference(encryptedSopsSecret, kubeSecretFromTemplate, r.Scheme)
	if err != nil {
		encryptedSopsSecret.Status.Message = "Setting controller ownership of the child secret error"
		r.Status().Update(context.Background(), encryptedSopsSecret)

		r.Log.Info(
			"Setting controller ownership of the child secret error",
			"sopssecret", req.NamespacedName,
			"error", err,
		)

		return nil, true
	}

	return kubeSecretFromTemplate, false
}

func (r *SopsSecretReconciler) isSecretSuspended(
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret, req ctrl.Request) bool {

	// Return early if SopsSecret object is suspended.
	if encryptedSopsSecret.Spec.Suspend {
		r.Log.Info(
			"Reconciliation is suspended for this object",
			"sopssecret", req.NamespacedName,
		)

		encryptedSopsSecret.Status.Message = "Reconciliation is suspended"
		r.Status().Update(context.Background(), encryptedSopsSecret)

		return true
	}

	return false
}

func (r *SopsSecretReconciler) getEncryptedSopsSecret(
	ctx context.Context, req ctrl.Request) (*gitopssecretsnappcloudiov1alpha1.SopsSecret, bool, error) {

	encryptedSopsSecret := &gitopssecretsnappcloudiov1alpha1.SopsSecret{}

	err := r.Get(ctx, req.NamespacedName, encryptedSopsSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info(
				"Request object not found, could have been deleted after reconcile request",
				"sopssecret",
				req.NamespacedName,
			)
			return nil, true, nil
		}

		// Error reading the object - requeue the request.
		r.Log.Info(
			"Error reading the object",
			"sopssecret",
			req.NamespacedName,
		)
		return nil, true, err
	}
	return encryptedSopsSecret, false, nil
}

// checks if the annotation equals to "true", and it's case sensitive
func isAnnotatedToBeManaged(secret *corev1.Secret) bool {
	return secret.Annotations[gitopssecretsnappcloudiov1alpha1.SopsSecretManagedAnnotation] == "true"
}

// SetupWithManager sets up the controller with the Manager.
func (r *SopsSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Set logging level
	sopslogging.SetLevel(logrus.InfoLevel)

	// Set logrus logs to be discarded
	for k := range sopslogging.Loggers {
		sopslogging.Loggers[k].Out = ioutil.Discard
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopssecretsnappcloudiov1alpha1.SopsSecret{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// createKubeSecretFromTemplate returns new Kubernetes secret object, created from decrypted SopsSecret Template
func createKubeSecretFromTemplate(
	sopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	stringData *map[string]string,
	logger logr.Logger,
) (*corev1.Secret, error) {
	kubeSecretType := "Opaque"
	labels := cloneMap(sopsSecret.Labels)
	annotations := cloneMap(sopsSecret.Annotations)

	logger.Info("Processing",
		"sopssecret", fmt.Sprintf("%s.%s.%s", sopsSecret.Kind, sopsSecret.APIVersion, sopsSecret.Name),
		"type", kubeSecretType,
		"namespace", sopsSecret.Namespace,
		"templateItem", fmt.Sprintf("secret/%s", sopsSecret.Name),
	)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        sopsSecret.Name,
			Namespace:   sopsSecret.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       corev1.SecretType(kubeSecretType),
		StringData: *stringData,
	}
	return secret, nil
}

func cloneMap(oldMap map[string]string) map[string]string {
	newMap := make(map[string]string)

	for key, value := range oldMap {
		newMap[key] = value
	}

	return newMap
}

func getSecretType(templateSecretType string) corev1.SecretType {
	var kubeSecretType corev1.SecretType

	switch templateSecretType {
	case "kubernetes.io/service-account-token":
		kubeSecretType = corev1.SecretTypeServiceAccountToken
	case "kubernetes.io/dockercfg":
		kubeSecretType = corev1.SecretTypeDockercfg
	case "kubernetes.io/dockerconfigjson":
		kubeSecretType = corev1.SecretTypeDockerConfigJson
	case "kubernetes.io/basic-auth":
		kubeSecretType = corev1.SecretTypeBasicAuth
	case "kubernetes.io/ssh-auth":
		kubeSecretType = corev1.SecretTypeSSHAuth
	case "kubernetes.io/tls":
		kubeSecretType = corev1.SecretTypeTLS
	case "bootstrap.kubernetes.io/token":
		kubeSecretType = corev1.SecretTypeBootstrapToken
	default:
		kubeSecretType = corev1.SecretTypeOpaque
	}

	return kubeSecretType
}

// decryptSopsSecretInstance decrypts spec.secretTemplates
func decryptSopsSecretInstance(
	encryptedSopsSecret *gitopssecretsnappcloudiov1alpha1.SopsSecret,
	logger logr.Logger,
	passphrase string,
) (*gitopssecretsnappcloudiov1alpha1.SopsSecret, error) {
	sopsSecretAsBytes, err := json.Marshal(encryptedSopsSecret)
	if err != nil {
		logger.Info(
			"Failed to convert encrypted sops secret to bytes[]",
			"sopssecret", fmt.Sprintf("%s/%s", encryptedSopsSecret.Namespace, encryptedSopsSecret.Name),
			"error", err,
		)
		return nil, err
	}

	decryptedSopsSecretAsBytes, err := customDecryptData(sopsSecretAsBytes, "json", passphrase)
	if err != nil {
		logger.Info(
			"Failed to Decrypt encrypted sops secret decryptedSopsSecret",
			"sopssecret", fmt.Sprintf("%s/%s", encryptedSopsSecret.Namespace, encryptedSopsSecret.Name),
			"error", err,
		)
		return nil, err
	}

	decryptedSopsSecret := &gitopssecretsnappcloudiov1alpha1.SopsSecret{}
	err = json.Unmarshal(decryptedSopsSecretAsBytes, &decryptedSopsSecret)
	if err != nil {
		logger.Info(
			"Failed to Unmarshal decrypted sops secret decryptedSopsSecret",
			"sopssecret", fmt.Sprintf("%s/%s", encryptedSopsSecret.Namespace, encryptedSopsSecret.Name),
			"error", err,
		)
		return nil, err
	}

	return decryptedSopsSecret, nil
}

// Data is a helper that takes encrypted data and a format string,
// decrypts the data and returns its cleartext in an []byte.
// The format string can be `json`, `yaml`, `dotenv` or `binary`.
// If the format string is empty, binary format is assumed.
// NOTE: this function is taken from sops code and adjusted
//       to ignore mac, as CR will always be mutated in k8s
func customDecryptData(data []byte, format string, passphrase string) (cleartext []byte, err error) {
	// Initialize a Sops JSON store
	var store sops.Store

	switch format {
	case "json":
		store = &sopsjson.Store{}
	case "yaml":
		store = &sopsyaml.Store{}
	case "dotenv":
		store = &sopsdotenv.Store{}
	default:
		store = &sopsjson.BinaryStore{}
	}

	// Load SOPS file and access the data key
	tree, err := store.LoadEncryptedFile(data)
	if err != nil {
		return nil, err
	}

	key, err := GetDataKeyCustom(tree.Metadata, passphrase)
	if userErr, ok := err.(sops.UserError); ok {
		err = fmt.Errorf(userErr.UserError())
	}
	if err != nil {
		return nil, err
	}

	// Decrypt the tree
	cipher := sopsaes.NewCipher()
	_, err = tree.Decrypt(key, cipher)
	if err != nil {
		return nil, err
	}

	return store.EmitPlainFile(tree.Branches)
}
