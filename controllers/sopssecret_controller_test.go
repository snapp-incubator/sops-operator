package controllers_test

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gitopssecretsnappcloudiov1alpha1 "github.com/snapp-incubator/sops-operator/api/v1alpha1"
	controller "github.com/snapp-incubator/sops-operator/controllers"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"path/filepath"
	"time"
)

var (
	exampleGPGFilePath       = filepath.Join("..", "config", "pgp-test-key", "gpgkey.yaml")
	exampleGPGFileUnsafePath = filepath.Join("..", "config", "pgp-test-key", "gpgkey_unsafe0.yaml")
	exampleFilePath          = filepath.Join("..", "config", "pgp-test-key", "example.enc.yaml")
)

var _ = Describe("", func() {
	TestGPGKeyObj := &gitopssecretsnappcloudiov1alpha1.GPGKey{}
	TestGPGKeyObjUnsafe := &gitopssecretsnappcloudiov1alpha1.GPGKey{}
	TestSopsSecretObj := &gitopssecretsnappcloudiov1alpha1.SopsSecret{}

	BeforeEach(func() {
		content, err := ioutil.ReadFile(exampleGPGFileUnsafePath)
		Expect(err).Should(BeNil())

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(content, nil, nil)
		TestGPGKeyObjUnsafe = obj.(*gitopssecretsnappcloudiov1alpha1.GPGKey)
		Expect(err).Should(BeNil())
	})

	BeforeEach(func() {
		content, err := ioutil.ReadFile(exampleGPGFilePath)
		Expect(err).Should(BeNil())

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(content, nil, nil)
		TestGPGKeyObj = obj.(*gitopssecretsnappcloudiov1alpha1.GPGKey)
		Expect(err).Should(BeNil())
	})

	BeforeEach(func() {
		content, err := ioutil.ReadFile(exampleFilePath)
		Expect(err).Should(BeNil())

		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(content, nil, nil)
		TestSopsSecretObj = obj.(*gitopssecretsnappcloudiov1alpha1.SopsSecret)
		Expect(err).Should(BeNil())
	})

	const (
		GPGKeyRefName       = "gpgkey-sample"
		GPGKeyRefUnsafeName = "gpgkey-unsafe"
		SopsSecretName      = "example-secret"
		SopsSecretNamespace = "default"

		timeout   = time.Second * 360
		sleepTime = time.Second * 3
	)

	Context("When Creating Correctly Defined SopsSecret Object", func() {
		It("Should fail to Decrypt SopsSecret", func() {
			By("Using wrong password on unsafe GPGKey")
			ctx := context.Background()
			Expect(controller.K8sClient.Create(ctx, TestGPGKeyObjUnsafe)).To(Succeed())
			time.Sleep(sleepTime)

			gpgkey := &gitopssecretsnappcloudiov1alpha1.GPGKey{}
			err := controller.K8sClient.Get(ctx, types.NamespacedName{Namespace: SopsSecretNamespace, Name: GPGKeyRefUnsafeName}, gpgkey)
			Expect(err).To(BeNil())

			By("By creating a new SopsSecret")
			Expect(controller.K8sClient.Create(ctx, TestSopsSecretObj)).To(Succeed())
			time.Sleep(sleepTime)

			By("By checking data values")
			testSecret := &corev1.Secret{}
			targetSecretNamespacedName := &types.NamespacedName{Namespace: SopsSecretNamespace, Name: SopsSecretName}
			Expect(controller.K8sClient.Get(ctx, *targetSecretNamespacedName, testSecret)).NotTo(Succeed())
			time.Sleep(sleepTime)

			By("Deleting the SopsSecret Object")
			Expect(controller.K8sClient.Delete(ctx, TestSopsSecretObj)).To(Succeed())
			time.Sleep(sleepTime)
		})

		It("Should Succeed to Create SopsSecret", func() {
			By("Importing it's content from file and Creating GPGKey")
			ctx := context.Background()
			Expect(controller.K8sClient.Create(ctx, TestGPGKeyObj)).To(Succeed())
			time.Sleep(sleepTime)

			gpgkey := &gitopssecretsnappcloudiov1alpha1.GPGKey{}
			err := controller.K8sClient.Get(ctx, types.NamespacedName{Namespace: SopsSecretNamespace, Name: GPGKeyRefName}, gpgkey)
			Expect(err).To(BeNil())

			By("By creating a new SopsSecret")
			Expect(controller.K8sClient.Create(ctx, TestSopsSecretObj)).To(Succeed())
			time.Sleep(sleepTime)

			By("By checking data values")
			testSecret := &corev1.Secret{}
			targetSecretNamespacedName := &types.NamespacedName{Namespace: SopsSecretNamespace, Name: SopsSecretName}
			Expect(controller.K8sClient.Get(ctx, *targetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(string(testSecret.Data["data-name0"])).To(Equal("data-value0"))
			Expect(string(testSecret.Data["data-name1"])).To(Equal("data-value1"))

			By("By removing secret template from SopsSecret must remove managed k8s secret")
			testSecret = &corev1.Secret{}
			targetSecretNamespacedName = &types.NamespacedName{Namespace: SopsSecretNamespace, Name: SopsSecretName}
			Expect(controller.K8sClient.Get(ctx, *targetSecretNamespacedName, testSecret)).To(Succeed())
			Expect(controller.K8sClient.Delete(ctx, testSecret)).To(Succeed())
			time.Sleep(10 * time.Second)
			secretsList := &corev1.SecretList{}
			Expect(controller.K8sClient.List(ctx, secretsList)).To(Succeed())
			newTestSecret := &corev1.Secret{}
			Expect(controller.K8sClient.Get(ctx, *targetSecretNamespacedName, newTestSecret)).To(Succeed())
		}, float64(timeout))
	})
})
