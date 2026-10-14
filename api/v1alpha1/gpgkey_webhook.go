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
	passwordValidator "github.com/go-passwd/validator"
	"github.com/snapp-incubator/sops-operator/lang"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

// log is for logging in this package.
var (
	gpgKeyLog                        = logf.Log.WithName("gpgkey-resource")
	gPGKeyArmoredPrivateKeyMinLength = 1024
)

func (r *GPGKey) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(&gpgKeyDefaulter{}).
		WithValidator(&gpgKeyValidator{}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-gitopssecret-snappcloud-io-v1alpha1-gpgkey,mutating=true,failurePolicy=fail,sideEffects=None,groups=gitopssecret.snappcloud.io,resources=gpgkeys,verbs=create;update,versions=v1alpha1,name=mgpgkey.kb.io,admissionReviewVersions=v1

type gpgKeyDefaulter struct{}

var _ admission.Defaulter[*GPGKey] = &gpgKeyDefaulter{}

// Default implements admission.Defaulter so a webhook will be registered for the type
func (d *gpgKeyDefaulter) Default(_ context.Context, r *GPGKey) error {
	gpgKeyLog.Info("default", "name", r.Name)
	return nil
}

//+kubebuilder:webhook:path=/validate-gitopssecret-snappcloud-io-v1alpha1-gpgkey,mutating=false,failurePolicy=fail,sideEffects=None,groups=gitopssecret.snappcloud.io,resources=gpgkeys,verbs=create;update,versions=v1alpha1,name=vgpgkey.kb.io,admissionReviewVersions=v1

type gpgKeyValidator struct{}

var _ admission.Validator[*GPGKey] = &gpgKeyValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type
func (v *gpgKeyValidator) ValidateCreate(_ context.Context, r *GPGKey) (admission.Warnings, error) {
	gpgKeyLog.Info("validate create", "name", r.Name)
	return nil, r.ValidateGPGKey()
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type
func (v *gpgKeyValidator) ValidateUpdate(_ context.Context, _, r *GPGKey) (admission.Warnings, error) {
	gpgKeyLog.Info("validate update", "name", r.Name)
	return nil, r.ValidateGPGKey()
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type
func (v *gpgKeyValidator) ValidateDelete(_ context.Context, r *GPGKey) (admission.Warnings, error) {
	gpgKeyLog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *GPGKey) ValidateGPGKey() error {
	passValidObj := GetPasswordValidator()
	if err := passValidObj.Validate(r.Spec.Passphrase); err != nil {
		return err
	}
	trimedArmoredPrivateKey := strings.TrimSpace(r.Spec.ArmoredPrivateKey)
	if len(trimedArmoredPrivateKey) < gPGKeyArmoredPrivateKeyMinLength {
		return errors.New(lang.ErrGPGKeySpecArmoredPrivateKeyLength)
	} else if strings.HasPrefix(trimedArmoredPrivateKey, "----") || strings.HasSuffix(trimedArmoredPrivateKey, "----") {
		return errors.New(lang.ErrGPGKeySpecArmoredPrivateKeyPrefixSuffix)
	}
	return nil
}

func GetPasswordValidator() *passwordValidator.Validator {
	return passwordValidator.New(
		passwordValidator.MinLength(14, errors.New(lang.ErrGPGKeySpecPassphraseLength)),
		passwordValidator.MaxLength(100, errors.New(lang.ErrGPGKeySpecPassphraseLength)),
		passwordValidator.CommonPassword(errors.New(lang.ErrGPGKeySpecPassphraseCommon)),
	)
}
