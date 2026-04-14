# This script bootstraps a complete local Shukra development cluster on Windows.
# It exists so a new user can go from repository checkout to a working operator
# and sample AppEnvironment with one PowerShell command.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File .\hack\bootstrap-local.ps1
#
# The script intentionally performs the following workflow:
# 1. Start Docker Desktop and wait for the engine
# 2. Create or recreate a kind cluster
# 3. Build the operator image
# 4. Load the image into kind
# 5. Install cert-manager
# 6. Install or upgrade the Shukra Helm chart
# 7. Apply the working basic example
# 8. Wait for the sample Deployment to become available

[CmdletBinding()]
param(
    [string]$ClusterName = "shukra",
    [string]$ImageName = "shukra-operator:dev",
    [string]$CertManagerVersion = "v1.17.4",
    [string]$OperatorNamespace = "shukra-system",
    [string]$ExampleNamespace = "default",
    [switch]$RecreateCluster,
    [switch]$SkipExample
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

# Add common Windows install locations so the script works on fresh shells where
# winget or manual tool installs are present but not yet visible in PATH.
$pathCandidates = @(
    "D:\tools\bin",
    "D:\tools\go\bin",
    "$env:LOCALAPPDATA\Microsoft\WinGet\Links",
    "C:\Program Files\Docker\Docker\resources\bin",
    "C:\Program Files\Docker\Docker"
)
foreach ($candidate in $pathCandidates) {
    if ($candidate -and (Test-Path $candidate) -and ($env:PATH -notlike "*$candidate*")) {
        $env:PATH = "$candidate;$env:PATH"
    }
}

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Require-Command {
    param([string]$Name)
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if (-not $cmd) {
        throw "Required command '$Name' was not found in PATH. Install it first, then rerun this script."
    }
    return $cmd.Source
}

function Wait-Docker {
    param([string]$DockerCli)
    for ($i = 0; $i -lt 60; $i++) {
        try {
            & $DockerCli info | Out-Null
            return
        } catch {
            Start-Sleep -Seconds 5
        }
    }
    throw "Docker did not become ready within the expected time."
}

function Ensure-Namespace {
    param([string]$Namespace)
    kubectl create namespace $Namespace --dry-run=client -o yaml | kubectl apply -f - | Out-Null
}

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

$dockerCli = "C:\Program Files\Docker\Docker\resources\bin\docker.exe"
$dockerDesktop = "C:\Program Files\Docker\Docker\Docker Desktop.exe"

if (-not (Test-Path $dockerCli)) {
    $dockerCli = Require-Command "docker"
}

if (Test-Path $dockerDesktop) {
    Write-Step "Starting Docker Desktop"
    Start-Process -FilePath $dockerDesktop | Out-Null
}

Write-Step "Waiting for Docker engine"
Wait-Docker -DockerCli $dockerCli

$null = Require-Command "kind"
$null = Require-Command "kubectl"
$null = Require-Command "helm"
$null = Require-Command "git"

Write-Step "Checking kind cluster state"
$clusters = kind get clusters
$clusterExists = $clusters -contains $ClusterName

if ($clusterExists -and $RecreateCluster) {
    Write-Step "Deleting existing kind cluster '$ClusterName'"
    kind delete cluster --name $ClusterName
    $clusterExists = $false
}

if (-not $clusterExists) {
    Write-Step "Creating kind cluster '$ClusterName'"
    kind create cluster --name $ClusterName --image kindest/node:v1.29.2
} else {
    Write-Step "Using existing kind cluster '$ClusterName'"
    kubectl cluster-info --context "kind-$ClusterName" | Out-Null
}

Write-Step "Building operator image '$ImageName'"
& $dockerCli build -t $ImageName .

Write-Step "Loading operator image into kind"
kind load docker-image $ImageName --name $ClusterName

Write-Step "Ensuring namespaces"
Ensure-Namespace -Namespace "cert-manager"
Ensure-Namespace -Namespace $OperatorNamespace

Write-Step "Installing cert-manager $CertManagerVersion"
helm repo add jetstack https://charts.jetstack.io 2>$null | Out-Null
helm repo update | Out-Null
helm upgrade --install cert-manager jetstack/cert-manager `
    --namespace cert-manager `
    --version $CertManagerVersion `
    --set crds.enabled=true `
    --wait `
    --timeout 10m

Write-Step "Waiting for cert-manager Pods"
kubectl wait --for=condition=Ready pods --all -n cert-manager --timeout=600s | Out-Null

Write-Step "Installing Shukra Operator chart"
helm upgrade --install shukra-operator charts/shukra-operator `
    -n $OperatorNamespace `
    --set image.repository=shukra-operator `
    --set image.tag=dev `
    --set image.pullPolicy=IfNotPresent `
    --set leaderElection.namespace=$OperatorNamespace `
    --wait `
    --timeout 10m

if (-not $SkipExample) {
    Write-Step "Applying basic AppEnvironment example"
    kubectl apply -f .\examples\basic.yaml

    Write-Step "Waiting for basic app rollout"
    kubectl rollout status deployment/basic-app-deployment -n $ExampleNamespace --timeout=300s
}

Write-Step "Bootstrap complete"
Write-Host "Cluster context: kind-$ClusterName"
Write-Host "Operator namespace: $OperatorNamespace"
if (-not $SkipExample) {
    Write-Host "Sample environment: basic-app"
    Write-Host ""
    Write-Host "Useful follow-up commands:"
    Write-Host "  kubectl get appenvironment basic-app -n $ExampleNamespace -o yaml"
    Write-Host "  kubectl get deploy,svc,cm,pods -n $ExampleNamespace"
    Write-Host "  kubectl logs -n $OperatorNamespace deploy/shukra-operator"
}
