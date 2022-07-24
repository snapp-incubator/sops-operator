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

package v1alpha1

import (
	"fmt"
	"github.com/snapp-incubator/sops-operator/lang"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var gpgkeylog = logf.Log.WithName("gpgkey-resource")

func (r *GPGKey) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-gitopssecret-snappcloud-io-v1alpha1-gpgkey,mutating=true,failurePolicy=fail,sideEffects=None,groups=gitopssecret.snappcloud.io,resources=gpgkeys,verbs=create;update,versions=v1alpha1,name=mgpgkey.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &GPGKey{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *GPGKey) Default() {
	gpgkeylog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-gitopssecret-snappcloud-io-v1alpha1-gpgkey,mutating=false,failurePolicy=fail,sideEffects=None,groups=gitopssecret.snappcloud.io,resources=gpgkeys,verbs=create;update,versions=v1alpha1,name=vgpgkey.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &GPGKey{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *GPGKey) ValidateCreate() error {
	gpgkeylog.Info("validate create", "name", r.Name)
	if err := r.ValidateGPGKey(); err != nil {
		return err
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *GPGKey) ValidateUpdate(old runtime.Object) error {
	gpgkeylog.Info("validate update", "name", r.Name)
	if err := r.ValidateGPGKey(); err != nil {
		return err
	}

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *GPGKey) ValidateDelete() error {
	gpgkeylog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *GPGKey) ValidateGPGKey() error {
	if len(r.Spec.Passphrase) < 4 {
		return fmt.Errorf(lang.ErrGPGKeySpecPassphraseLength)
	}
	if r.Spec.ArmoredPrivateKey == "" {
		return fmt.Errorf(lang.ErrGPGKeySpecArmoredPrivateKeyLength)
	}
	return nil
}
