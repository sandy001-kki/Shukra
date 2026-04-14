# API Reference

## Packages
- [apps.shukra.io/v1alpha1](#appsshukraiov1alpha1)
- [apps.shukra.io/v1beta1](#appsshukraiov1beta1)


## apps.shukra.io/v1alpha1

This file implements spoke conversion from v1alpha1 to the v1beta1 hub and
back. It exists because the two versions have structural differences for
secret references and network policy configuration.

This file defines the legacy v1alpha1 AppEnvironment schema. It exists so
clusters can serve older clients while storing data in v1beta1 through the
conversion webhook.

This file registers the served v1alpha1 API group/version. It exists so the
operator can support upgrades from an older schema through explicit conversion.

### Resource Types
- [AppEnvironment](#appenvironment)
- [AppEnvironmentList](#appenvironmentlist)



#### AppEnvironment







_Appears in:_
- [AppEnvironmentList](#appenvironmentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.shukra.io/v1alpha1` | | |
| `kind` _string_ | `AppEnvironment` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[AppEnvironmentSpec](#appenvironmentspec)_ |  |  |  |
| `status` _[AppEnvironmentStatus](#appenvironmentstatus)_ |  |  |  |


#### AppEnvironmentList









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.shukra.io/v1alpha1` | | |
| `kind` _string_ | `AppEnvironmentList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[AppEnvironment](#appenvironment) array_ |  |  |  |


#### AppEnvironmentSpec







_Appears in:_
- [AppEnvironment](#appenvironment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `app` _[AppSpec](#appspec)_ |  |  |  |
| `config` _[ConfigSpec](#configspec)_ |  |  |  |
| `service` _[ServiceSpec](#servicespec)_ |  |  |  |
| `ingress` _[IngressSpec](#ingressspec)_ |  |  |  |
| `database` _[DatabaseSpec](#databasespec)_ |  |  |  |
| `migration` _[MigrationSpec](#migrationspec)_ |  |  |  |
| `autoscaling` _[AutoscalingSpec](#autoscalingspec)_ |  |  |  |
| `backup` _[BackupSpec](#backupspec)_ |  |  |  |
| `restore` _[RestoreSpec](#restorespec)_ |  |  |  |
| `security` _[SecuritySpec](#securityspec)_ |  |  |  |
| `paused` _boolean_ |  |  |  |


#### AppEnvironmentStatus







_Appears in:_
- [AppEnvironment](#appenvironment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _string_ |  |  |  |
| `observedGeneration` _integer_ |  |  |  |
| `url` _string_ |  |  |  |
| `childResources` _[ChildResources](#childresources)_ |  |  |  |
| `lastError` _string_ |  |  |  |
| `failureCount` _integer_ |  |  |  |
| `lastAppliedMigrationID` _string_ |  |  |  |
| `lastProcessedRestoreNonce` _string_ |  |  |  |
| `lastSuccessfulReconcileTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#time-v1-meta)_ |  |  |  |
| `lastAppliedSpecHash` _string_ |  |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#condition-v1-meta) array_ |  |  |  |


#### AppSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ |  |  |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#pullpolicy-v1-core)_ |  |  |  |
| `replicas` _integer_ |  |  |  |
| `containerPort` _integer_ |  |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#envvar-v1-core) array_ |  |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#envfromsource-v1-core) array_ |  |  |  |
| `secretRefs` _string array_ |  |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#resourcerequirements-v1-core)_ |  |  |  |
| `strategy` _[DeploymentStrategy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#deploymentstrategy-v1-apps)_ |  |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#probe-v1-core)_ |  |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#probe-v1-core)_ |  |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#probe-v1-core)_ |  |  |  |
| `serviceAccountName` _string_ |  |  |  |


#### AutoscalingSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `minReplicas` _integer_ |  |  |  |
| `maxReplicas` _integer_ |  |  |  |
| `targetCPUUtilizationPercentage` _integer_ |  |  |  |
| `targetMemoryUtilizationPercentage` _integer_ |  |  |  |


#### BackupSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `schedule` _string_ |  |  |  |
| `destination` _string_ |  |  |  |
| `retentionPolicy` _string_ |  |  |  |
| `suspend` _boolean_ |  |  |  |


#### ChildResources







_Appears in:_
- [AppEnvironmentStatus](#appenvironmentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deploymentName` _string_ |  |  |  |
| `serviceName` _string_ |  |  |  |
| `configMapName` _string_ |  |  |  |
| `hpaName` _string_ |  |  |  |
| `ingressName` _string_ |  |  |  |
| `migrationJobName` _string_ |  |  |  |
| `restoreJobName` _string_ |  |  |  |
| `backupCronJobName` _string_ |  |  |  |
| `networkPolicyName` _string_ |  |  |  |
| `pdbName` _string_ |  |  |  |


#### ConfigSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _object (keys:string, values:string)_ |  |  |  |


#### DatabaseSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `mode` _string_ |  |  |  |
| `secretRef` _string_ |  |  |  |
| `schemaName` _string_ |  |  |  |


#### IngressSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `host` _string_ |  |  |  |
| `path` _string_ |  |  |  |
| `pathType` _string_ |  |  |  |
| `className` _string_ |  |  |  |
| `tlsSecretName` _string_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `allowSharedIngressHost` _boolean_ |  |  |  |


#### MigrationSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `image` _string_ |  |  |  |
| `command` _string array_ |  |  |  |
| `args` _string array_ |  |  |  |
| `migrationID` _string_ |  |  |  |
| `backoffLimit` _integer_ |  |  |  |
| `activeDeadlineSeconds` _integer_ |  |  |  |


#### PDBSpec







_Appears in:_
- [SecuritySpec](#securityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |


#### RestoreSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `image` _string_ |  |  |  |
| `source` _string_ |  |  |  |
| `triggerNonce` _string_ |  |  |  |
| `mode` _string_ |  |  |  |
| `targetRevision` _string_ |  |  |  |


#### SecuritySpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `networkPolicy` _boolean_ |  |  |  |
| `podDisruptionBudget` _[PDBSpec](#pdbspec)_ |  |  |  |


#### ServiceSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `type` _[ServiceType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#servicetype-v1-core)_ |  |  |  |
| `port` _integer_ |  |  |  |
| `targetPort` _integer_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |



## apps.shukra.io/v1beta1

This file defines the v1beta1 storage version of AppEnvironment. It exists as
the canonical API contract that the operator reconciles and persists.

This file registers the canonical v1beta1 API group/version. It exists so
controller-runtime and the Kubernetes API machinery can encode, decode, and
store AppEnvironment objects using the storage version.

### Resource Types
- [AppEnvironment](#appenvironment)
- [AppEnvironmentList](#appenvironmentlist)



#### AppEnvironment







_Appears in:_
- [AppEnvironmentList](#appenvironmentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.shukra.io/v1beta1` | | |
| `kind` _string_ | `AppEnvironment` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[AppEnvironmentSpec](#appenvironmentspec)_ |  |  |  |
| `status` _[AppEnvironmentStatus](#appenvironmentstatus)_ |  |  |  |


#### AppEnvironmentList









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `apps.shukra.io/v1beta1` | | |
| `kind` _string_ | `AppEnvironmentList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[AppEnvironment](#appenvironment) array_ |  |  |  |


#### AppEnvironmentSpec







_Appears in:_
- [AppEnvironment](#appenvironment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `app` _[AppSpec](#appspec)_ |  |  |  |
| `config` _[ConfigSpec](#configspec)_ |  |  |  |
| `service` _[ServiceSpec](#servicespec)_ |  |  |  |
| `ingress` _[IngressSpec](#ingressspec)_ |  |  |  |
| `database` _[DatabaseSpec](#databasespec)_ |  |  |  |
| `migration` _[MigrationSpec](#migrationspec)_ |  |  |  |
| `autoscaling` _[AutoscalingSpec](#autoscalingspec)_ |  |  |  |
| `backup` _[BackupSpec](#backupspec)_ |  |  |  |
| `restore` _[RestoreSpec](#restorespec)_ |  |  |  |
| `security` _[SecuritySpec](#securityspec)_ |  |  |  |
| `paused` _boolean_ |  |  |  |


#### AppEnvironmentStatus







_Appears in:_
- [AppEnvironment](#appenvironment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `phase` _string_ |  |  |  |
| `observedGeneration` _integer_ |  |  |  |
| `url` _string_ |  |  |  |
| `childResources` _[ChildResources](#childresources)_ |  |  |  |
| `lastError` _string_ |  |  |  |
| `failureCount` _integer_ |  |  |  |
| `lastAppliedMigrationID` _string_ |  |  |  |
| `lastProcessedRestoreNonce` _string_ |  |  |  |
| `lastSuccessfulReconcileTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#time-v1-meta)_ |  |  |  |
| `lastAppliedSpecHash` _string_ |  |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#condition-v1-meta) array_ |  |  |  |


#### AppSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _string_ |  |  |  |
| `imagePullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#pullpolicy-v1-core)_ |  |  |  |
| `replicas` _integer_ |  |  |  |
| `containerPort` _integer_ |  |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#envvar-v1-core) array_ |  |  |  |
| `envFrom` _[EnvFromSource](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#envfromsource-v1-core) array_ |  |  |  |
| `secretRefs` _[SecretRef](#secretref) array_ |  |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#resourcerequirements-v1-core)_ |  |  |  |
| `strategy` _[DeploymentStrategy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#deploymentstrategy-v1-apps)_ |  |  |  |
| `livenessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#probe-v1-core)_ |  |  |  |
| `readinessProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#probe-v1-core)_ |  |  |  |
| `startupProbe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#probe-v1-core)_ |  |  |  |
| `serviceAccountName` _string_ |  |  |  |


#### AutoscalingSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `minReplicas` _integer_ |  |  |  |
| `maxReplicas` _integer_ |  |  |  |
| `targetCPUUtilizationPercentage` _integer_ |  |  |  |
| `targetMemoryUtilizationPercentage` _integer_ |  |  |  |


#### BackupSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `schedule` _string_ |  |  |  |
| `destination` _string_ |  |  |  |
| `retentionPolicy` _string_ |  |  |  |
| `suspend` _boolean_ |  |  |  |


#### ChildResources







_Appears in:_
- [AppEnvironmentStatus](#appenvironmentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deploymentName` _string_ |  |  |  |
| `serviceName` _string_ |  |  |  |
| `configMapName` _string_ |  |  |  |
| `hpaName` _string_ |  |  |  |
| `ingressName` _string_ |  |  |  |
| `migrationJobName` _string_ |  |  |  |
| `restoreJobName` _string_ |  |  |  |
| `backupCronJobName` _string_ |  |  |  |
| `networkPolicyName` _string_ |  |  |  |
| `pdbName` _string_ |  |  |  |


#### ConfigSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `data` _object (keys:string, values:string)_ |  |  |  |


#### DatabaseSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `mode` _string_ |  |  |  |
| `secretRef` _string_ |  |  |  |
| `schemaName` _string_ |  |  |  |


#### IngressSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `host` _string_ |  |  |  |
| `path` _string_ |  |  |  |
| `pathType` _[PathType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#pathtype-v1-networking)_ |  |  |  |
| `className` _string_ |  |  |  |
| `tlsSecretName` _string_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |
| `allowSharedIngressHost` _boolean_ |  |  |  |


#### MigrationSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `image` _string_ |  |  |  |
| `command` _string array_ |  |  |  |
| `args` _string array_ |  |  |  |
| `migrationID` _string_ |  |  |  |
| `backoffLimit` _integer_ |  |  |  |
| `activeDeadlineSeconds` _integer_ |  |  |  |


#### NetworkPolicySpec







_Appears in:_
- [SecuritySpec](#securityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ingressRules` _[NetworkPolicyIngressRule](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#networkpolicyingressrule-v1-networking) array_ |  |  |  |
| `egressRules` _[NetworkPolicyEgressRule](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#networkpolicyegressrule-v1-networking) array_ |  |  |  |


#### PDBSpec







_Appears in:_
- [SecuritySpec](#securityspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `minAvailable` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#intorstring-intstr-util)_ |  |  |  |
| `maxUnavailable` _[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#intorstring-intstr-util)_ |  |  |  |


#### RestoreSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `image` _string_ |  |  |  |
| `source` _string_ |  |  |  |
| `triggerNonce` _string_ |  |  |  |
| `mode` _string_ |  |  |  |
| `targetRevision` _string_ |  |  |  |


#### SecretRef







_Appears in:_
- [AppSpec](#appspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `mountAs` _string_ |  |  |  |
| `mountPath` _string_ |  |  |  |


#### SecuritySpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `networkPolicy` _[NetworkPolicySpec](#networkpolicyspec)_ |  |  |  |
| `podDisruptionBudget` _[PDBSpec](#pdbspec)_ |  |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ |  |  |  |
| `containerSecurityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ |  |  |  |


#### ServiceSpec







_Appears in:_
- [AppEnvironmentSpec](#appenvironmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `type` _[ServiceType](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#servicetype-v1-core)_ |  |  |  |
| `port` _integer_ |  |  |  |
| `targetPort` _integer_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |


