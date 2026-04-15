# This script generates a Shukra-specific AI dataset from repository documents,
# examples, and other grounded project sources. It exists so future retrieval or
# fine-tuning work starts from repo truth instead of ad-hoc copied text.

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$datasetRoot = Join-Path $repoRoot "ai\datasets"
$rawDir = Join-Path $datasetRoot "raw"
$generatedDir = Join-Path $datasetRoot "generated"

New-Item -ItemType Directory -Force -Path $rawDir | Out-Null
New-Item -ItemType Directory -Force -Path $generatedDir | Out-Null

$sources = @(
    "README.md",
    "docs\beginner-guide.md",
    "docs\getting-started.md",
    "docs\learning-path.md",
    "docs\cli.md",
    "docs\architecture.md",
    "docs\tenancy.md",
    "docs\troubleshooting.md",
    "docs\migration-restore-walkthrough.md",
    "examples\basic.yaml",
    "examples\ingress.yaml",
    "examples\autoscaling.yaml",
    "examples\migration.yaml",
    "examples\restore.yaml",
    "examples\paused.yaml"
)

$manifest = @()
$records = New-Object System.Collections.Generic.List[object]

foreach ($relativePath in $sources) {
    $fullPath = Join-Path $repoRoot $relativePath
    if (-not (Test-Path $fullPath)) {
        throw "Required AI source file not found: $relativePath"
    }

    $content = Get-Content -Raw -Path $fullPath
    $targetPath = Join-Path $rawDir (($relativePath -replace "[\\/]", "__"))
    Set-Content -Path $targetPath -Value $content

    $manifest += [pscustomobject]@{
        path       = $relativePath
        bytes      = (Get-Item $fullPath).Length
        extractedAt = (Get-Date).ToString("o")
    }

    $records.Add([pscustomobject]@{
        type   = "source_chunk"
        source = $relativePath
        prompt = "Summarize the purpose of $relativePath for a Shukra user."
        answer = ($content.Substring(0, [Math]::Min($content.Length, 2400)))
    })
}

$records.Add([pscustomobject]@{
    type   = "qa"
    source = "README.md"
    prompt = "What is Shukra Operator?"
    answer = "Shukra Operator is a production-grade Kubernetes Operator that provisions and manages full application environments from a single AppEnvironment custom resource."
})

$records.Add([pscustomobject]@{
    type   = "qa"
    source = "docs/cli.md"
    prompt = "How do I open the English-first Shukra assistant in PowerShell?"
    answer = "Run `shukra chat` for the interactive assistant or `shukra chat --message ""status basic-app""` for a one-shot request."
})

$records.Add([pscustomobject]@{
    type   = "qa"
    source = "docs/troubleshooting.md"
    prompt = "Why might a restore not trigger?"
    answer = "A restore job is only created when the restore trigger nonce changes. Reusing the same nonce should not create a new restore job."
})

$records.Add([pscustomobject]@{
    type   = "qa"
    source = "docs/tenancy.md"
    prompt = "Can Shukra reference secrets across namespaces?"
    answer = "No. Shukra treats the namespace as the tenant boundary and requires secret references to stay in the same namespace."
})

$manifest | ConvertTo-Json -Depth 4 | Set-Content -Path (Join-Path $generatedDir "source-manifest.json")
$records | ForEach-Object { $_ | ConvertTo-Json -Compress -Depth 6 } | Set-Content -Path (Join-Path $generatedDir "shukra-assistant-dataset.jsonl")

Write-Host "Generated AI dataset artifacts:"
Write-Host "  $(Join-Path $generatedDir 'source-manifest.json')"
Write-Host "  $(Join-Path $generatedDir 'shukra-assistant-dataset.jsonl')"
