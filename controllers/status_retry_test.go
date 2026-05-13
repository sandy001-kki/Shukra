package controllers

import (
	"context"
	"fmt"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1beta1 "github.com/sandy001-kki/Shukra/api/v1beta1"
)

func TestPersistStatusRetriesConflict(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := appsv1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	env := &appsv1beta1.AppEnvironment{
		ObjectMeta: metav1.ObjectMeta{Name: "retry", Namespace: "default"},
		Spec:       appsv1beta1.AppEnvironmentSpec{App: appsv1beta1.AppSpec{Image: "nginx:1.27"}},
	}
	baseClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(env).WithObjects(env).Build()
	reconciler := &AppEnvironmentReconciler{Client: &conflictOnceClient{Client: baseClient}, Scheme: scheme}

	env.Status.Phase = appsv1beta1.PhaseRunning
	if err := reconciler.persistStatus(context.Background(), env); err != nil {
		t.Fatalf("persistStatus returned error: %v", err)
	}

	current := &appsv1beta1.AppEnvironment{}
	if err := baseClient.Get(context.Background(), client.ObjectKey{Name: "retry", Namespace: "default"}, current); err != nil {
		t.Fatal(err)
	}
	if current.Status.Phase != appsv1beta1.PhaseRunning {
		t.Fatalf("expected phase %q after retry, got %q", appsv1beta1.PhaseRunning, current.Status.Phase)
	}
}

type conflictOnceClient struct {
	client.Client
	status conflictOnceStatusWriter
}

func (c *conflictOnceClient) Status() client.SubResourceWriter {
	c.status.SubResourceWriter = c.Client.Status()
	return &c.status
}

type conflictOnceStatusWriter struct {
	client.SubResourceWriter
	conflicted bool
}

func (w *conflictOnceStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	if !w.conflicted {
		w.conflicted = true
		return apierrors.NewConflict(schema.GroupResource{Group: "apps.shukra.io", Resource: "appenvironments"}, obj.GetName(), fmt.Errorf("forced conflict"))
	}
	return w.SubResourceWriter.Patch(ctx, obj, patch, opts...)
}
