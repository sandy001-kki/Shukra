// This file tests the English-first Shukra chat parser and doctor helpers. It
// exists so the assistant-style CLI can evolve with confidence instead of
// depending only on manual testing.
package cli

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseChatIntentStatus(t *testing.T) {
	intent := parseChatIntent("status basic-app", "default")
	if intent.Action != "status" {
		t.Fatalf("expected status action, got %q", intent.Action)
	}
	if intent.Name != "basic-app" {
		t.Fatalf("expected basic-app name, got %q", intent.Name)
	}
	if intent.Namespace != "default" {
		t.Fatalf("expected default namespace, got %q", intent.Namespace)
	}
}

func TestParseChatIntentList(t *testing.T) {
	intent := parseChatIntent("list environments", "default")
	if intent.Action != "list" {
		t.Fatalf("expected list action, got %q", intent.Action)
	}
	if intent.Target != "environments" {
		t.Fatalf("expected environments target, got %q", intent.Target)
	}
}

func TestParseChatIntentResources(t *testing.T) {
	intent := parseChatIntent("show resources for basic-app", "default")
	if intent.Action != "resources" {
		t.Fatalf("expected resources action, got %q", intent.Action)
	}
	if intent.Name != "basic-app" {
		t.Fatalf("expected basic-app name, got %q", intent.Name)
	}
}

func TestParseChatIntentDiagnoseOperator(t *testing.T) {
	intent := parseChatIntent("show operator status", "default")
	if intent.Action != "diagnose" {
		t.Fatalf("expected diagnose action, got %q", intent.Action)
	}
	if intent.Target != "operator" {
		t.Fatalf("expected operator target, got %q", intent.Target)
	}
}

func TestParseChatIntentInstallOCI(t *testing.T) {
	intent := parseChatIntent("install operator from oci version 0.2.3", "default")
	if intent.Action != "install" {
		t.Fatalf("expected install action, got %q", intent.Action)
	}
	if !intent.UseOCI {
		t.Fatal("expected OCI install mode to be enabled")
	}
	if intent.ChartVersion != "0.2.3" {
		t.Fatalf("expected chart version 0.2.3, got %q", intent.ChartVersion)
	}
}

func TestParseChatIntentRestore(t *testing.T) {
	intent := parseChatIntent("restore basic-app with nonce restore-001 image busybox:1.36 source s3://bucket/backup mode full", "default")
	if intent.Action != "restore" {
		t.Fatalf("expected restore action, got %q", intent.Action)
	}
	if intent.Name != "basic-app" || intent.TriggerNonce != "restore-001" || intent.Image != "busybox:1.36" || intent.Source != "s3://bucket/backup" || intent.RestoreMode != "full" {
		t.Fatalf("unexpected restore parse result: %#v", intent)
	}
}

func TestParseChatIntentUnknown(t *testing.T) {
	intent := parseChatIntent("sing me a song", "default")
	if intent.Action != "" {
		t.Fatalf("expected empty action for unknown command, got %q", intent.Action)
	}
	if intent.UnknownReason == "" {
		t.Fatal("expected unknown reason to be populated")
	}
}

func TestDisplayVersion(t *testing.T) {
	got := displayVersion("0.2.3", "abc123", "2026-04-15")
	want := "0.2.3 | abc123 | 2026-04-15"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestHasDoctorFailure(t *testing.T) {
	results := []doctorResult{
		{Name: "Docker CLI", Status: "ok"},
		{Name: "Kubernetes API", Status: "fail"},
	}
	if !hasDoctorFailure(results) {
		t.Fatal("expected doctor failure to be detected")
	}
}

func TestIsPodReady(t *testing.T) {
	pod := corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}
	if !isPodReady(pod) {
		t.Fatal("expected pod to be ready")
	}
}

func TestConditionStatus(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: "Ready", Status: metav1.ConditionTrue},
	}
	if got := conditionStatus(conditions, "Ready"); got != "True" {
		t.Fatalf("expected True, got %q", got)
	}
}
