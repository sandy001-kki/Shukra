// This file wraps the Kubernetes event recorder with typed helper methods. It
// exists to keep lifecycle event messages consistent across the controller.
package events

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type eventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
	Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{})
}

type Recorder struct {
	recorder eventRecorder
}

func New(recorder eventRecorder) Recorder {
	return Recorder{recorder: recorder}
}

func (r Recorder) RecordReconcileStarted(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "ReconcileStarted", "Reconciliation started")
}

func (r Recorder) RecordChildReconciled(obj runtime.Object, child string) {
	r.recorder.Eventf(obj, corev1.EventTypeNormal, "ChildReconciled", "%s created or updated", child)
}

func (r Recorder) RecordMigrationStarted(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "MigrationStarted", "Migration job created")
}

func (r Recorder) RecordMigrationSucceeded(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "MigrationSucceeded", "Migration job completed")
}

func (r Recorder) RecordMigrationFailed(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeWarning, "MigrationFailed", "Migration job failed")
}

func (r Recorder) RecordRestoreStarted(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "RestoreStarted", "Restore job created")
}

func (r Recorder) RecordRestoreSucceeded(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "RestoreSucceeded", "Restore job completed")
}

func (r Recorder) RecordRestoreFailed(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeWarning, "RestoreFailed", "Restore job failed")
}

func (r Recorder) RecordPaused(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "Paused", "Reconciliation paused by spec")
}

func (r Recorder) RecordUnpaused(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "Unpaused", "Reconciliation resumed")
}

func (r Recorder) RecordDeletionStarted(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "DeletionStarted", "Deletion and finalizer cleanup started")
}

func (r Recorder) RecordAionosIntentViolated(obj runtime.Object, intentType string) {
	r.recorder.Eventf(obj, corev1.EventTypeWarning, "AionosIntentViolated", "Intent %s is violated", intentType)
}

func (r Recorder) RecordAionosIntentSatisfied(obj runtime.Object, intentType string) {
	r.recorder.Eventf(obj, corev1.EventTypeNormal, "AionosIntentSatisfied", "Intent %s is satisfied", intentType)
}

func (r Recorder) RecordAionosPatchApplied(obj runtime.Object, patchID, appliedBy string) {
	r.recorder.Eventf(obj, corev1.EventTypeNormal, "AionosPatchApplied", "Patch %s applied by %s", patchID, appliedBy)
}

func (r Recorder) RecordAionosShadowCreated(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "AionosShadowCreated", "Shadow environment created")
}

func (r Recorder) RecordAionosShadowExpired(obj runtime.Object) {
	r.recorder.Event(obj, corev1.EventTypeNormal, "AionosShadowExpired", "Shadow environment TTL expired")
}
