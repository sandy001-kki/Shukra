# This script validates the AI workspace state for Shukra. It exists so
# contributors can quickly confirm that the data-preparation layer is populated
# before moving into retrieval or model-tuning work.

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot

$requiredPaths = @(
    "docs\ai-roadmap.md",
    "docs\ai-architecture.md",
    "ai\README.md",
    "ai\prompts\assistant-system.md",
    "ai\datasets\generated\source-manifest.json",
    "ai\datasets\generated\shukra-assistant-dataset.jsonl"
)

$missing = @()
foreach ($relativePath in $requiredPaths) {
    $fullPath = Join-Path $repoRoot $relativePath
    if (-not (Test-Path $fullPath)) {
        $missing += $relativePath
    }
}

if ($missing.Count -gt 0) {
    Write-Error ("AI readiness failed. Missing: " + ($missing -join ", "))
    exit 1
}

$datasetPath = Join-Path $repoRoot "ai\datasets\generated\shukra-assistant-dataset.jsonl"
$lineCount = (Get-Content $datasetPath).Count

if ($lineCount -lt 5) {
    Write-Error "AI readiness failed. Dataset JSONL has too few records."
    exit 1
}

Write-Host "AI readiness passed."
Write-Host "  Dataset lines: $lineCount"
Write-Host "  Required docs and prompt templates are present."
