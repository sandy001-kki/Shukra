// This file defines the legacy v1alpha1 AppEnvironment schema. It exists so
// clusters can serve older clients while storing data in v1beta1 through the
// conversion webhook.
package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type AppSpec struct {
	Image              string                      `json:"image"`
	ImagePullPolicy    corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Replicas           *int32                      `json:"replicas,omitempty"`
	ContainerPort      int32                       `json:"containerPort,omitempty"`
	Env                []corev1.EnvVar             `json:"env,omitempty"`
	EnvFrom            []corev1.EnvFromSource      `json:"envFrom,omitempty"`
	SecretRefs         []string                    `json:"secretRefs,omitempty"`
	Resources          corev1.ResourceRequirements `json:"resources,omitempty"`
	Strategy           appsv1.DeploymentStrategy   `json:"strategy,omitempty"`
	LivenessProbe      *corev1.Probe               `json:"livenessProbe,omitempty"`
	ReadinessProbe     *corev1.Probe               `json:"readinessProbe,omitempty"`
	StartupProbe       *corev1.Probe               `json:"startupProbe,omitempty"`
	ServiceAccountName string                      `json:"serviceAccountName,omitempty"`
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
	PathType               string            `json:"pathType,omitempty"`
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

type SecuritySpec struct {
	NetworkPolicy bool `json:"networkPolicy,omitempty"`
	PodDisruptionBudget PDBSpec `json:"podDisruptionBudget,omitempty"`
}

type PDBSpec struct {
	Enabled bool `json:"enabled,omitempty"`
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
	Phase                       string             `json:"phase,omitempty"`
	ObservedGeneration          int64              `json:"observedGeneration,omitempty"`
	URL                         string             `json:"url,omitempty"`
	ChildResources              ChildResources     `json:"childResources,omitempty"`
	LastError                   string             `json:"lastError,omitempty"`
	FailureCount                int32              `json:"failureCount,omitempty"`
	LastAppliedMigrationID      string             `json:"lastAppliedMigrationID,omitempty"`
	LastProcessedRestoreNonce   string             `json:"lastProcessedRestoreNonce,omitempty"`
	LastSuccessfulReconcileTime *metav1.Time       `json:"lastSuccessfulReconcileTime,omitempty"`
	LastAppliedSpecHash         string             `json:"lastAppliedSpecHash,omitempty"`
	Conditions                  []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
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
