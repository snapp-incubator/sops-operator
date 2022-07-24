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
	"bytes"
	"context"
	"github.com/go-logr/logr"
	gitopssecretsnappcloudiov1alpha1 "github.com/snapp-incubator/sops-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"os/exec"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"
)

var (
	GPGKeyImportedSuccessfully = "Imported"
	GPGKeyFailedToImport       = "Failed"
)

// GPGKeyReconciler reconciles a GPGKey object
type GPGKeyReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	Log          logr.Logger
	RequeueAfter int64
}

//+kubebuilder:rbac:groups=gitopssecret.snappcloud.io,resources=gpgkeys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gitopssecret.snappcloud.io,resources=gpgkeys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gitopssecret.snappcloud.io,resources=gpgkeys/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GPGKey object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *GPGKeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	loggerObj := log.FromContext(ctx)
	loggerObj.Info("strated gpgkey reconciler")

	gpgKey, finishLoop, err := r.getGPGKey(req)
	if finishLoop {
		return ctrl.Result{}, err
	}

	rescheduleReconcileLoop := r.importKey(req, gpgKey)
	if rescheduleReconcileLoop {
		return reconcile.Result{Requeue: true, RequeueAfter: time.Duration(r.RequeueAfter) * time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

func (r *GPGKeyReconciler) getGPGKey(req ctrl.Request) (*gitopssecretsnappcloudiov1alpha1.GPGKey, bool, error) {
	gpgKey := &gitopssecretsnappcloudiov1alpha1.GPGKey{}
	err := r.Get(context.Background(), req.NamespacedName, gpgKey)
	if err != nil {
		r.Log.Info("Couldn't get GPGKey obj", "gpgkey", req.NamespacedName)
		return gpgKey, true, err
	}
	return gpgKey, false, nil
}

func (r *GPGKeyReconciler) importKey(req ctrl.Request, gpgKey *gitopssecretsnappcloudiov1alpha1.GPGKey) bool {
	keyDirPath := filepath.Join("keys", req.Namespace)
	err := createKeyDirectories(keyDirPath)
	if err != nil {
		r.Log.Info("Couldn't create directory for gpgkey", "gpgkey", req.NamespacedName)
		gpgKey.Status.Message = GPGKeyFailedToImport
		r.Status().Update(context.Background(), gpgKey)
		return true
	}

	keyFullPath := filepath.Join(keyDirPath, req.Name+".gpg")
	err = createKeyFile(keyFullPath, gpgKey)
	if err != nil {
		r.Log.Info("Couldn't create file for gpgkey", "gpgkey", req.NamespacedName)
		gpgKey.Status.Message = GPGKeyFailedToImport
		r.Status().Update(context.Background(), gpgKey)
		return true
	}

	args := []string{
		"--batch",
		"--import",
		keyFullPath,
	}
	cmd := exec.Command(gpgBinary(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader(gpgKey.Spec.ArmoredPrivateKey)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		r.Log.Info("Couldn't import gpgkey", "gpgkey", req.NamespacedName, "stderr", stderr.String())
		gpgKey.Status.Message = GPGKeyFailedToImport
		r.Status().Update(context.Background(), gpgKey)
		return true
	}
	gpgKey.Status.Message = GPGKeyImportedSuccessfully
	r.Status().Update(context.Background(), gpgKey)
	return false
}

func createKeyDirectories(dirPath string) error {
	return os.MkdirAll(dirPath, os.ModePerm)
}

func createKeyFile(filePath string, gpgKey *gitopssecretsnappcloudiov1alpha1.GPGKey) error {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			_, err2 := os.Create(filePath)
			if err2 != nil {
				return err2
			}
			return writeFile(filePath, gpgKey)
		} else {
			return err
		}
	}
	return writeFile(filePath, gpgKey)
}

func writeFile(path string, gpgKey *gitopssecretsnappcloudiov1alpha1.GPGKey) error {
	// Open file using READ & WRITE permission.
	var file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	_, err = file.WriteString("-----BEGIN PGP PRIVATE KEY BLOCK-----\n\n" + strings.TrimSpace(gpgKey.Spec.ArmoredPrivateKey) + "\n-----END PGP PRIVATE KEY BLOCK-----")
	if err != nil {
		return err
	}

	err = file.Sync()
	if err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GPGKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopssecretsnappcloudiov1alpha1.GPGKey{}).
		Complete(r)
}
