package v1alpha1

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/snapp-incubator/sops-operator/lang"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("SopsSecret webhook", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		fooSopsSecretName          = "foo-sopssecret"
		fooSopsSecretNameSpace     = "default"
		fooSopsSecretGPGKeyRefName = "foo-gpgkey"
	)
	var (
		err error
		//sopssecret *SopsSecret
		ctx                     = context.Background()
		fooSopsSecretStringData = map[string]string{"fooStringDataKey": "fooStringDataValue"}
	)

	sopsSecretTypeMeta := metav1.TypeMeta{
		APIVersion: "gitopssecret.snappcloud.io/v1alpha1",
		Kind:       "SopsSecret",
	}

	foosopsSecretMeta := &SopsSecret{
		TypeMeta: sopsSecretTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      fooSopsSecretName,
			Namespace: fooSopsSecretNameSpace,
		},
	}

	AfterEach(func() {
		err = k8sClient.Delete(ctx, foosopsSecretMeta)
		if err != nil {
			Expect(errors.IsNotFound(err)).Should(BeTrue())
		}
	})

	Context("When creating a SopsSecret", func() {
		It("Should fail if stringData is empty", func() {
			By("Creating a SopsSecret with empty Spec.stringData")
			fooSopsSecretObj := &SopsSecret{
				TypeMeta:   foosopsSecretMeta.TypeMeta,
				ObjectMeta: foosopsSecretMeta.ObjectMeta,
				Spec: SopsSecretSpec{
					Suspend:       false,
					GPGKeyRefName: fooSopsSecretGPGKeyRefName,
					StringData:    map[string]string{},
				},
			}
			err = k8sClient.Create(ctx, fooSopsSecretObj)
			Expect(err).NotTo(BeNil())
			Expect(string(errors.ReasonForError(err))).Should(Equal(lang.ErrSopsSecretSpecNoData))
		})

		It("Should fail if gpg_key_ref_name is empty", func() {
			By("Creating a SopsSecret without Spec.GPGKeyRefName")
			fooSopsSecretObj := &SopsSecret{
				TypeMeta:   foosopsSecretMeta.TypeMeta,
				ObjectMeta: foosopsSecretMeta.ObjectMeta,
				Spec: SopsSecretSpec{
					Suspend:    false,
					StringData: fooSopsSecretStringData,
				},
			}
			err = k8sClient.Create(ctx, fooSopsSecretObj)
			Expect(err).NotTo(BeNil())
			Expect(string(errors.ReasonForError(err))).Should(Equal(lang.ErrSopsSecretSpecGPGKeyRefNameEmpty))
		})

		It("Should create if suspend is empty", func() {
			By("Creating a SopsSecret without Spec.suspend")
			barSopsSecretObj := &SopsSecret{
				TypeMeta:   foosopsSecretMeta.TypeMeta,
				ObjectMeta: foosopsSecretMeta.ObjectMeta,
				Spec: SopsSecretSpec{
					GPGKeyRefName: fooSopsSecretGPGKeyRefName,
					StringData:    fooSopsSecretStringData,
				},
			}
			err = k8sClient.Create(ctx, barSopsSecretObj)
			Expect(err).To(BeNil())
			sopsLookupKey := types.NamespacedName{Name: barSopsSecretObj.GetName(), Namespace: barSopsSecretObj.GetNamespace()}
			err = k8sClient.Get(ctx, sopsLookupKey, barSopsSecretObj)
			Expect(barSopsSecretObj.Spec.Suspend).To(BeFalse())
		})
	})
})
