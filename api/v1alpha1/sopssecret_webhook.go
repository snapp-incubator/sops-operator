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
	"context"
	"errors"
	"github.com/snapp-incubator/sops-operator/lang"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var sopssecretlog = logf.Log.WithName("sopssecret-resource")

func (r *SopsSecret) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(&sopsSecretDefaulter{}).
		WithValidator(&sopsSecretValidator{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-gitopssecret-snappcloud-io-v1alpha1-sopssecret,mutating=true,failurePolicy=fail,sideEffects=None,groups=gitopssecret.snappcloud.io,resources=sopssecrets,verbs=create;update,versions=v1alpha1,name=msopssecret.kb.io,admissionReviewVersions=v1

type sopsSecretDefaulter struct{}

var _ admission.Defaulter[*SopsSecret] = &sopsSecretDefaulter{}

// Default implements admission.Defaulter so a webhook will be registered for the type
func (d *sopsSecretDefaulter) Default(_ context.Context, r *SopsSecret) error {
	sopssecretlog.Info("default", "name", r.Name)
	return nil
}

//+kubebuilder:webhook:path=/validate-gitopssecret-snappcloud-io-v1alpha1-sopssecret,mutating=false,failurePolicy=fail,sideEffects=None,groups=gitopssecret.snappcloud.io,resources=sopssecrets,verbs=create;update,versions=v1alpha1,name=vsopssecret.kb.io,admissionReviewVersions=v1

type sopsSecretValidator struct{}

var _ admission.Validator[*SopsSecret] = &sopsSecretValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (v *sopsSecretValidator) ValidateCreate(_ context.Context, r *SopsSecret) (admission.Warnings, error) {
	sopssecretlog.Info("validate create", "name", r.Name)
	return nil, r.ValidateSopsSecret()
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (v *sopsSecretValidator) ValidateUpdate(_ context.Context, _, r *SopsSecret) (admission.Warnings, error) {
	sopssecretlog.Info("validate update", "name", r.Name)
	return nil, r.ValidateSopsSecret()
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (v *sopsSecretValidator) ValidateDelete(_ context.Context, r *SopsSecret) (admission.Warnings, error) {
	sopssecretlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *SopsSecret) ValidateSopsSecret() error {
	if r.Spec.GPGKeyRefName == "" {
		return errors.New(lang.ErrSopsSecretSpecGPGKeyRefNameEmpty)
	}
	if len(r.Spec.StringData) == 0 {
		return errors.New(lang.ErrSopsSecretSpecNoData)
	}
	return nil
}
