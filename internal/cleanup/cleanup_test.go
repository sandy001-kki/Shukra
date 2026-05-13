package cleanup

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDatabaseHookSkipsWhenSecretMissingBestEffort(t *testing.T) {
	c := fakeClient(t)
	hook := NewDatabaseHook(c)
	if err := hook.Cleanup(context.Background(), CleanupTarget{Name: "app", Namespace: "demo", DatabaseSecret: "missing"}); err != nil {
		t.Fatalf("expected missing database secret to be best-effort nil, got %v", err)
	}
}

func TestBackupHookDeletesExistingCronJob(t *testing.T) {
	cronJob := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: "app-backup", Namespace: "demo"}}
	c := fakeClient(t, cronJob)
	hook := NewBackupHook(c)
	if err := hook.Cleanup(context.Background(), CleanupTarget{Name: "app", Namespace: "demo", BackupDestination: "s3://bucket/app"}); err != nil {
		t.Fatalf("backup cleanup returned error: %v", err)
	}
	current := &batchv1.CronJob{}
	if err := c.Get(context.Background(), client.ObjectKey{Name: "app-backup", Namespace: "demo"}, current); err == nil {
		t.Fatalf("expected backup CronJob to be deleted")
	}
}

func TestDNSHookNoopsWithoutExternalDNSAnnotation(t *testing.T) {
	ingress := &networkingIngressForTest
	c := fakeClient(t, ingress)
	hook := NewDNSHook(c)
	if err := hook.Cleanup(context.Background(), CleanupTarget{Name: "app", Namespace: "demo", IngressHost: "app.example.com"}); err != nil {
		t.Fatalf("DNS cleanup returned error: %v", err)
	}
}

func fakeClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

var networkingIngressForTest = networkingv1.Ingress{
	ObjectMeta: metav1.ObjectMeta{Name: "app-ingress", Namespace: "demo"},
}
