package v1alpha1

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/snapp-incubator/sops-operator/lang"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

var _ = Describe("GPGKey webhook", func() {
	const (
		fooGPGKeyName      = "foo-gpgkey"
		fooGPGKeyNamespace = "default"

		wrongPassword0   = ""
		wrongPassword1   = "password"
		correctPassword0 = "qwerP@ssw0rdasdf12345"

		wrongArmoredKey0 = ""
		wrongArmoredKey1 = "fake-data"
	)
	var (
		correctArmoredKey1 = strings.Repeat("a", 1024)
		err                error
		ctx                = context.Background()
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

	Context("Testing Password Validator", func() {
		It("Should return error on weak passwords", func() {
			passwordValidatorObj := GetPasswordValidator()

			By("testing on weak passwords")
			weakPasswords := []string{
				"hello",
				"password",
				"P@ssw0rd",
				"1234",
				"helloworld",
			}
			for _, pass := range weakPasswords {
				err := passwordValidatorObj.Validate(pass)
				Expect(err).NotTo(BeNil())
			}

			By("testing on strong passwords")
			strongPasswords := []string{
				"qwerP@ssw0rdasdf12345",
				"qwedfsswzzrdas:df1W3U5",
			}
			for _, pass := range strongPasswords {
				err := passwordValidatorObj.Validate(pass)
				Expect(err).To(BeNil())
			}
		})
	})

	Context("When creating a GPGKey", func() {
		It("Should fail if passphrase has problem", func() {
			By("Creating a password with length of zero")
			fooGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: correctArmoredKey1,
					Passphrase:        wrongPassword0,
				},
			}
			err = k8sClient.Create(ctx, fooGPGKeyObj)
			Expect(err).NotTo(BeNil())
			Expect(string(errors.ReasonForError(err))).Should(Equal(lang.ErrGPGKeySpecPassphraseLength))

			By("Creating a password with length lower than minimum expected")
			barGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: correctArmoredKey1,
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
					ArmoredPrivateKey: correctArmoredKey1,
					Passphrase:        correctPassword0,
				},
			}
			err = k8sClient.Create(ctx, fooGPGKeyObj)
			Expect(err).To(BeNil())
		})
	})

	Context("When creating a GPGKey", func() {
		It("Should fail if armoredPrivateKey is not ok", func() {
			By("Creating a GPGKey with empty Private Key")
			fooGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: wrongArmoredKey0,
					Passphrase:        correctPassword0,
				},
			}
			err = k8sClient.Create(ctx, fooGPGKeyObj)
			Expect(err).NotTo(BeNil())

			By("Creating a GPGKey with length of Private Key lower than the minimum expected")
			barGPGKeyObj := &GPGKey{
				TypeMeta:   fooGPGKeyMeta.TypeMeta,
				ObjectMeta: fooGPGKeyMeta.ObjectMeta,
				Spec: GPGKeySpec{
					ArmoredPrivateKey: wrongArmoredKey1,
					Passphrase:        correctPassword0,
				},
			}
			err = k8sClient.Create(ctx, barGPGKeyObj)
			Expect(err).NotTo(BeNil())
		})
	})
})
