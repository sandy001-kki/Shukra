// This file defines the v1beta1 storage version of AppEnvironment. It exists as
// the canonical API contract that the operator reconciles and persists.
package v1beta1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ConditionSpecValid       = "SpecValid"
	ConditionConfigReady     = "ConfigReady"
	ConditionServiceReady    = "ServiceReady"
	ConditionDeploymentReady = "DeploymentReady"
	ConditionDatabaseReady   = "DatabaseReady"
	ConditionMigrationReady  = "MigrationReady"
	ConditionIngressReady    = "IngressReady"
	ConditionBackupReady     = "BackupReady"
	ConditionRestoreReady    = "RestoreReady"
	ConditionReady           = "Ready"
	ConditionPaused          = "Paused"
)

const (
	PhasePending     = "Pending"
	PhaseConfiguring = "Configuring"
	PhaseRunning     = "Running"
	PhaseDegraded    = "Degraded"
	PhaseFailed      = "Failed"
	PhasePaused      = "Paused"
	PhaseRestoring   = "Restoring"
	PhaseDeleting    = "Deleting"
)

type AppEnvironmentSpec struct {
	App         AppSpec         `json:"app"`
	Config      ConfigSpec      `json:"config,omitempty"`
	Service     ServiceSpec     `json:"service,omitempty"`
	Ingress     IngressSpec     `json:"ingress,omitempty"`
	Database    DatabaseSpec    `json:"database,omitempty"`
	Migration   MigrationSpec   `json:"migration,omitempty"`
	Autoscaling AutoscalingSpec `json:"autoscaling,omitempty"`
	Backup      BackupSpec      `json:"backup,omitempty"`
	Restore     RestoreSpec     `json:"restore,omitempty"`
	Security    SecuritySpec    `json:"security,omitempty"`
	Paused      bool            `json:"paused,omitempty"`
}

type SecretRef struct {
	Name      string `json:"name"`
	MountAs   string `json:"mountAs,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
}

type AppSpec struct {
	Image                    string                      `json:"image"`
	ImagePullPolicy          corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Replicas                 *int32                      `json:"replicas,omitempty"`
	ContainerPort            int32                       `json:"containerPort,omitempty"`
	Env                      []corev1.EnvVar             `json:"env,omitempty"`
	EnvFrom                  []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	SecretRefs               []SecretRef                 `json:"secretRefs,omitempty"`
	Resources                corev1.ResourceRequirements `json:"resources,omitempty"`
	Strategy                 appsv1.DeploymentStrategy   `json:"strategy,omitempty"`
	LivenessProbe            *corev1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe           *corev1.Probe               `json:"readinessProbe,omitempty"`
	StartupProbe             *corev1.Probe               `json:"startupProbe,omitempty"`
	ServiceAccountName       string                      `json:"serviceAccountName,omitempty"`
}

type ConfigSpec struct {
	Data map[string]string `json:"data,omitempty"`
}

type ServiceSpec struct {
	Enabled     *bool             `json:"enabled,omitempty"`
	Type        corev1.ServiceType `json:"type,omitempty"`
	Port        int32             `json:"port,omitempty"`
	TargetPort  int32             `json:"targetPort,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type IngressSpec struct {
	Enabled                bool              `json:"enabled,omitempty"`
	Host                   string            `json:"host,omitempty"`
	Path                   string            `json:"path,omitempty"`
	PathType               *networkingv1.PathType `json:"pathType,omitempty"`
	ClassName              string            `json:"className,omitempty"`
	TLSSecretName          string            `json:"tlsSecretName,omitempty"`
	Annotations            map[string]string `json:"annotations,omitempty"`
	AllowSharedIngressHost bool              `json:"allowSharedIngressHost,omitempty"`
}

type DatabaseSpec struct {
	Enabled    bool   `json:"enabled,omitempty"`
	Mode       string `json:"mode,omitempty"`
	SecretRef  string `json:"secretRef,omitempty"`
	SchemaName string `json:"schemaName,omitempty"`
}

type MigrationSpec struct {
	Enabled               bool     `json:"enabled,omitempty"`
	Image                 string   `json:"image,omitempty"`
	Command               []string `json:"command,omitempty"`
	Args                  []string `json:"args,omitempty"`
	MigrationID           string   `json:"migrationID,omitempty"`
	BackoffLimit          *int32   `json:"backoffLimit,omitempty"`
	ActiveDeadlineSeconds *int64   `json:"activeDeadlineSeconds,omitempty"`
}

type AutoscalingSpec struct {
	Enabled                           bool   `json:"enabled,omitempty"`
	MinReplicas                       *int32 `json:"minReplicas,omitempty"`
	MaxReplicas                       int32  `json:"maxReplicas,omitempty"`
	TargetCPUUtilizationPercentage    *int32 `json:"targetCPUUtilizationPercentage,omitempty"`
	TargetMemoryUtilizationPercentage *int32 `json:"targetMemoryUtilizationPercentage,omitempty"`
}

type BackupSpec struct {
	Enabled         bool   `json:"enabled,omitempty"`
	Schedule        string `json:"schedule,omitempty"`
	Destination     string `json:"destination,omitempty"`
	RetentionPolicy string `json:"retentionPolicy,omitempty"`
	Suspend         bool   `json:"suspend,omitempty"`
}

type RestoreSpec struct {
	Enabled        bool   `json:"enabled,omitempty"`
	Image          string `json:"image,omitempty"`
	Source         string `json:"source,omitempty"`
	TriggerNonce   string `json:"triggerNonce,omitempty"`
	Mode           string `json:"mode,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
}

type NetworkPolicySpec struct {
	IngressRules []networkingv1.NetworkPolicyIngressRule `json:"ingressRules,omitempty"`
	EgressRules  []networkingv1.NetworkPolicyEgressRule  `json:"egressRules,omitempty"`
}

type PDBSpec struct {
	Enabled        bool                `json:"enabled,omitempty"`
	MinAvailable   *intstr.IntOrString `json:"minAvailable,omitempty"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

type SecuritySpec struct {
	NetworkPolicy            NetworkPolicySpec       `json:"networkPolicy,omitempty"`
	PodDisruptionBudget      PDBSpec                 `json:"podDisruptionBudget,omitempty"`
	PodSecurityContext       *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`
	ContainerSecurityContext *corev1.SecurityContext `json:"containerSecurityContext,omitempty"`
}

type ChildResources struct {
	DeploymentName    string `json:"deploymentName,omitempty"`
	ServiceName       string `json:"serviceName,omitempty"`
	ConfigMapName     string `json:"configMapName,omitempty"`
	HPAName           string `json:"hpaName,omitempty"`
	IngressName       string `json:"ingressName,omitempty"`
	MigrationJobName  string `json:"migrationJobName,omitempty"`
	RestoreJobName    string `json:"restoreJobName,omitempty"`
	BackupCronJobName string `json:"backupCronJobName,omitempty"`
	NetworkPolicyName string `json:"networkPolicyName,omitempty"`
	PDBName           string `json:"pdbName,omitempty"`
}

type AppEnvironmentStatus struct {
	Phase                      string             `json:"phase,omitempty"`
	ObservedGeneration         int64              `json:"observedGeneration,omitempty"`
	URL                        string             `json:"url,omitempty"`
	ChildResources             ChildResources     `json:"childResources,omitempty"`
	LastError                  string             `json:"lastError,omitempty"`
	FailureCount               int32              `json:"failureCount,omitempty"`
	LastAppliedMigrationID     string             `json:"lastAppliedMigrationID,omitempty"`
	LastProcessedRestoreNonce  string             `json:"lastProcessedRestoreNonce,omitempty"`
	LastSuccessfulReconcileTime *metav1.Time      `json:"lastSuccessfulReconcileTime,omitempty"`
	LastAppliedSpecHash        string             `json:"lastAppliedSpecHash,omitempty"`
	Conditions                 []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,shortName=appenv
type AppEnvironment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppEnvironmentSpec   `json:"spec,omitempty"`
	Status AppEnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AppEnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppEnvironment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppEnvironment{}, &AppEnvironmentList{})
}

func (*AppEnvironment) Hub() {}

func (in *AppEnvironment) SpecHash() string {
	payload, _ := json.Marshal(in.Spec)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (in *AppEnvironment) EffectiveReplicas() int32 {
	if in.Spec.App.Replicas == nil {
		return 2
	}
	return *in.Spec.App.Replicas
}

func (in *AppEnvironment) EffectiveContainerPort() int32 {
	if in.Spec.App.ContainerPort == 0 {
		return 8080
	}
	return in.Spec.App.ContainerPort
}

func (in *AppEnvironment) EffectiveServiceEnabled() bool {
	if in.Spec.Service.Enabled == nil {
		return true
	}
	return *in.Spec.Service.Enabled
}

func (in *AppEnvironment) ImageTag() string {
	parts := strings.Split(in.Spec.App.Image, ":")
	if len(parts) < 2 {
		return "latest"
	}
	return parts[len(parts)-1]
}

func (in *AppEnvironment) Labels(component string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       in.Name,
		"app.kubernetes.io/managed-by": "shukra-operator",
		"app.kubernetes.io/component":  component,
		"app.kubernetes.io/version":    in.ImageTag(),
		"apps.shukra.io/environment":   in.Name,
	}
}

func (in *AppEnvironment) ServicePort() int32 {
	if in.Spec.Service.Port == 0 {
		return 80
	}
	return in.Spec.Service.Port
}

func (in *AppEnvironment) ServiceTargetPort() int32 {
	if in.Spec.Service.TargetPort == 0 {
		return in.EffectiveContainerPort()
	}
	return in.Spec.Service.TargetPort
}

func (in *AppEnvironment) MigrationJobName() string {
	return fmt.Sprintf("%s-migration-%s", in.Name, strings.ToLower(in.Spec.Migration.MigrationID))
}

func (in *AppEnvironment) RestoreJobName() string {
	return fmt.Sprintf("%s-restore-%s", in.Name, strings.ToLower(in.Spec.Restore.TriggerNonce))
}
