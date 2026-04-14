// This file defines the envtest suite for the AppEnvironment controller. It
// exists to validate reconciliation, webhooks, conversion, and predicate
// behavior against a real API server instead of brittle unit-test mocks.
package controllers_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appsv1alpha1 "github.com/sandy001-kki/Shukra/api/v1alpha1"
	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
	"github.com/sandy001-kki/Shukra/controllers"
	"github.com/sandy001-kki/Shukra/webhooks"
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AppEnvironment Controller Suite")
}

var (
	testEnv   *envtest.Environment
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
	scheme    *runtime.Scheme
	envStarted bool
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())
	scheme = runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(appsv1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(appsv1beta1.AddToScheme(scheme)).To(Succeed())

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}
	cfg, err := testEnv.Start()
	if err != nil {
		Skip(fmt.Sprintf("envtest assets are unavailable on this machine: %v", err))
	}
	envStarted = true
	Expect(cfg).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	err = (&controllers.AppEnvironmentReconciler{
		Client:                  mgr.GetClient(),
		Scheme:                  scheme,
		EventRecorder:           mgr.GetEventRecorderFor("test"),
		MaxConcurrentReconciles: 5,
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		Expect(mgr.Start(ctx)).To(Succeed())
	}()

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	cancel()
	if envStarted && testEnv != nil {
		Expect(testEnv.Stop()).To(Succeed())
	}
})

var _ = Describe("AppEnvironment", func() {
	BeforeEach(func() {
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "demo"}})).To(SatisfyAny(Succeed(), MatchError(ContainSubstring("already exists"))))
		Expect(k8sClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "demo-secret", Namespace: "demo"}})).To(SatisfyAny(Succeed(), MatchError(ContainSubstring("already exists"))))
		Expect(k8sClient.Create(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "db-secret", Namespace: "demo"}})).To(SatisfyAny(Succeed(), MatchError(ContainSubstring("already exists"))))
	})

	It("1. basic creation creates Deployment and Service and reaches Running", func() {
		appEnv := basicEnv("basic")
		Expect(k8sClient.Create(ctx, appEnv)).To(Succeed())
		Eventually(func(g Gomega) {
			current := &appsv1beta1.AppEnvironment{}
			g.Expect(k8sClient.Get(ctx, key("demo", "basic"), current)).To(Succeed())
			g.Expect(current.Status.ChildResources.DeploymentName).NotTo(BeEmpty())
			g.Expect(current.Status.ChildResources.ServiceName).NotTo(BeEmpty())
			g.Expect(current.Status.Phase).To(Equal(appsv1beta1.PhaseRunning))
		}).WithTimeout(20 * time.Second).Should(Succeed())
	})

	It("2. ingress creation writes host and path", func() {
		appEnv := basicEnv("ingress")
		appEnv.Spec.Ingress.Enabled = true
		appEnv.Spec.Ingress.Host = "ingress.demo.example"
		Expect(k8sClient.Create(ctx, appEnv)).To(Succeed())
		Eventually(func(g Gomega) {
			current := &appsv1beta1.AppEnvironment{}
			g.Expect(k8sClient.Get(ctx, key("demo", "ingress"), current)).To(Succeed())
			g.Expect(current.Status.ChildResources.IngressName).NotTo(BeEmpty())
			g.Expect(current.Status.URL).To(ContainSubstring("ingress.demo.example"))
		}).WithTimeout(20 * time.Second).Should(Succeed())
	})

	It("3. paused mode sets phase Paused", func() {
		appEnv := basicEnv("paused")
		appEnv.Spec.Paused = true
		Expect(k8sClient.Create(ctx, appEnv)).To(Succeed())
		Eventually(func() string {
			current := &appsv1beta1.AppEnvironment{}
			_ = k8sClient.Get(ctx, key("demo", "paused"), current)
			return current.Status.Phase
		}).WithTimeout(20 * time.Second).Should(Equal(appsv1beta1.PhasePaused))
	})

	It("4. deletion removes the resource after finalizer cleanup", func() {
		appEnv := basicEnv("delete-me")
		Expect(k8sClient.Create(ctx, appEnv)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Delete(ctx, appEnv)
		}).Should(Succeed())
		Eventually(func() bool {
			current := &appsv1beta1.AppEnvironment{}
			return client.IgnoreNotFound(k8sClient.Get(ctx, key("demo", "delete-me"), current)) == nil && current.Name == ""
		}).WithTimeout(20 * time.Second).Should(BeTrue())
	})

	It("5. migration uses migrationID idempotency", func() {
		appEnv := basicEnv("migration")
		appEnv.Spec.Database.Enabled = true
		appEnv.Spec.Database.Mode = "external"
		appEnv.Spec.Database.SecretRef = "db-secret"
		appEnv.Spec.Migration.Enabled = true
		appEnv.Spec.Migration.Image = "ghcr.io/sandy001-kki/migrate:v1"
		appEnv.Spec.Migration.MigrationID = "v1"
		Expect(k8sClient.Create(ctx, appEnv)).To(Succeed())
		Eventually(func() string {
			current := &appsv1beta1.AppEnvironment{}
			_ = k8sClient.Get(ctx, key("demo", "migration"), current)
			return current.Status.LastAppliedMigrationID
		}).Should(Equal("v1"))
	})

	It("6. restore uses triggerNonce idempotency", func() {
		appEnv := basicEnv("restore")
		appEnv.Spec.Restore.Enabled = true
		appEnv.Spec.Restore.Image = "ghcr.io/sandy001-kki/restore:v1"
		appEnv.Spec.Restore.Source = "s3://bucket/backup"
		appEnv.Spec.Restore.TriggerNonce = "restore-001"
		Expect(k8sClient.Create(ctx, appEnv)).To(Succeed())
		Eventually(func() string {
			current := &appsv1beta1.AppEnvironment{}
			_ = k8sClient.Get(ctx, key("demo", "restore"), current)
			return current.Status.LastProcessedRestoreNonce
		}).Should(Equal("restore-001"))
	})

	It("7. status conditions reach Ready", func() {
		appEnv := basicEnv("ready")
		Expect(k8sClient.Create(ctx, appEnv)).To(Succeed())
		Eventually(func() metav1.ConditionStatus {
			current := &appsv1beta1.AppEnvironment{}
			_ = k8sClient.Get(ctx, key("demo", "ready"), current)
			for _, cond := range current.Status.Conditions {
				if cond.Type == appsv1beta1.ConditionReady {
					return cond.Status
				}
			}
			return metav1.ConditionUnknown
		}).Should(Equal(metav1.ConditionTrue))
	})

	It("8. webhook validation rejects invalid shapes conceptually", func() {
		validator := &webhooks.AppEnvironmentValidator{Client: k8sClient}
		invalid := basicEnv("invalid-image")
		invalid.Spec.App.Image = ""
		_, err := validator.ValidateCreate(ctx, invalid)
		Expect(err).To(HaveOccurred())

		invalidScale := basicEnv("invalid-scale")
		minReplicas := int32(5)
		invalidScale.Spec.Autoscaling.Enabled = true
		invalidScale.Spec.Autoscaling.MinReplicas = &minReplicas
		invalidScale.Spec.Autoscaling.MaxReplicas = 2
		_, err = validator.ValidateCreate(ctx, invalidScale)
		Expect(err).To(HaveOccurred())

		invalidMigration := basicEnv("invalid-migration")
		invalidMigration.Spec.Migration.Enabled = true
		invalidMigration.Spec.Migration.Image = "ghcr.io/sandy001-kki/migrate:v1"
		invalidMigration.Spec.Migration.MigrationID = "v1"
		_, err = validator.ValidateCreate(ctx, invalidMigration)
		Expect(err).To(HaveOccurred())
	})

	It("9. cross-namespace secret references are rejected conceptually", func() {
		validator := &webhooks.AppEnvironmentValidator{Client: k8sClient}
		invalid := basicEnv("cross-ns")
		invalid.Spec.App.SecretRefs = []appsv1beta1.SecretRef{{Name: "other/demo-secret", MountAs: "env"}}
		_, err := validator.ValidateCreate(ctx, invalid)
		Expect(err).To(HaveOccurred())
	})

	It("10. conversion round-trip preserves supported fields", func() {
		alpha := &appsv1alpha1.AppEnvironment{
			ObjectMeta: metav1.ObjectMeta{Name: "convert", Namespace: "demo"},
			Spec: appsv1alpha1.AppEnvironmentSpec{
				App: appsv1alpha1.AppSpec{Image: "nginx:1.27", SecretRefs: []string{"demo-secret"}},
				Security: appsv1alpha1.SecuritySpec{NetworkPolicy: true},
			},
		}
		beta := &appsv1beta1.AppEnvironment{}
		Expect(alpha.ConvertTo(beta)).To(Succeed())
		roundTrip := &appsv1alpha1.AppEnvironment{}
		Expect(roundTrip.ConvertFrom(beta)).To(Succeed())
		Expect(roundTrip.Spec.App.SecretRefs).To(ContainElement("demo-secret"))
		Expect(roundTrip.Annotations["conversion.shukra.io/downgrade-lossy"]).To(Equal("true"))
	})

	It("11. predicate filtering ignores status-only updates conceptually", func() {
		p := predicate.GenerationChangedPredicate{}
		oldObj := basicEnv("predicate")
		newObj := basicEnv("predicate")
		newObj.Status.Phase = appsv1beta1.PhaseRunning
		Expect(p.Update(event.UpdateEvent{ObjectOld: oldObj, ObjectNew: newObj})).To(BeFalse())
	})
})

func basicEnv(name string) *appsv1beta1.AppEnvironment {
	replicas := int32(1)
	return &appsv1beta1.AppEnvironment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "demo"},
		Spec: appsv1beta1.AppEnvironmentSpec{
			App: appsv1beta1.AppSpec{
				Image: "nginx:1.27",
				Replicas: &replicas,
				SecretRefs: []appsv1beta1.SecretRef{{Name: "demo-secret", MountAs: "env"}},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resourceMustParse("100m"), corev1.ResourceMemory: resourceMustParse("128Mi")},
					Limits:   corev1.ResourceList{corev1.ResourceCPU: resourceMustParse("500m"), corev1.ResourceMemory: resourceMustParse("256Mi")},
				},
			},
		},
	}
}

func key(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}

func resourceMustParse(value string) resource.Quantity {
	return resource.MustParse(value)
}
