package v1alpha1

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/snapp-incubator/sops-operator/lang"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GPGKey webhook", func() {
	const (
		fooGPGKeyName      = "foo-gpgkey"
		fooGPGKeyNamespace = "default"

		wrongPassword0   = ""
		wrongPassword1   = "a"
		correctPassword0 = "test2"

		armoredKey = "fake-data"
	)
	var (
		err error
		ctx = context.Background()
	)

	gpgKeyTypeMeta := metav1.TypeMeta{
		APIVersion: "gitopssecret.snappcloud.io/v1alpha1",
		Kind:       "GPGKey",
	}

	fooGPGKeyMeta := &GPGKey{
		TypeMeta: gpgKeyTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      fooGPGKeyName,
			Namespace: fooGPGKeyNamespace,
		},
	}

	AfterEach(func() {
		err = k8sClient.Delete(ctx, fooGPGKeyMeta)
		if err != nil {
			Expect(errors.IsNotFound(err)).Should(BeTrue())
		}
	})

	Context("When creating a GPGKey", func() {
		It("Should fail if passphrase has problem", func() {
			By("Creating a password with length of zero")
			fooGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: armoredKey,
					Passphrase:        wrongPassword0,
				},
			}
			err = k8sClient.Create(ctx, fooGPGKeyObj)
			Expect(err).NotTo(BeNil())
			Expect(string(errors.ReasonForError(err))).Should(Equal(lang.ErrGPGKeySpecPassphraseLength))

			By("Creating a password with length of one")
			barGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: armoredKey,
					Passphrase:        wrongPassword1,
				},
			}
			err = k8sClient.Create(ctx, barGPGKeyObj)
			Expect(err).NotTo(BeNil())
			Expect(string(errors.ReasonForError(err))).Should(Equal(lang.ErrGPGKeySpecPassphraseLength))
		})

		It("Should fail if armored private key is empty", func() {
			By("Creating a GPGKey with empty ArmoredPrivateKey")
			fooGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: "",
					Passphrase:        correctPassword0,
				},
			}
			err = k8sClient.Create(ctx, fooGPGKeyObj)
			Expect(err).NotTo(BeNil())
			Expect(string(errors.ReasonForError(err))).Should(Equal(lang.ErrGPGKeySpecArmoredPrivateKeyLength))
		})

		It("Should create if passphrase and armored key are ok", func() {
			By("Creating a GPGKey with proper passphrase length and proper key")
			fooGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: armoredKey,
					Passphrase:        correctPassword0,
				},
			}
			err = k8sClient.Create(ctx, fooGPGKeyObj)
			Expect(err).To(BeNil())
		})
	})
})
