# This script renders plain text into a terminal-style PNG image.
# It exists so repository docs can include lightweight screenshots generated
# directly from command output without needing external design tools.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File .\hack\render-terminal-png.ps1 `
#     -InputPath .\tmp\status.txt `
#     -OutputPath .\docs\assets\status.png `
#     -Title "Shukra demo"

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$InputPath,
    [Parameter(Mandatory = $true)]
    [string]$OutputPath,
    [string]$Title = "Terminal Output"
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing

$text = Get-Content -Raw -LiteralPath $InputPath
$lines = @($Title, "", $text -split "`r?`n")

$font = New-Object System.Drawing.Font("Consolas", 16)
$titleFont = New-Object System.Drawing.Font("Segoe UI Semibold", 16)
$padding = 32
$lineHeight = 28
$titleHeight = 36

$bitmap = New-Object System.Drawing.Bitmap(1600, 1200)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::ClearTypeGridFit

$maxWidth = 0
foreach ($line in $lines) {
    $measureFont = if ($line -eq $Title) { $titleFont } else { $font }
    $size = $graphics.MeasureString($line, $measureFont)
    if ($size.Width -gt $maxWidth) {
        $maxWidth = [Math]::Ceiling($size.Width)
    }
}

$width = [Math]::Max(900, $maxWidth + ($padding * 2))
$height = [Math]::Max(400, ($lines.Count * $lineHeight) + ($padding * 2) + 24)

$graphics.Dispose()
$bitmap.Dispose()

$bitmap = New-Object System.Drawing.Bitmap($width, $height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
$graphics.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::ClearTypeGridFit
$graphics.Clear([System.Drawing.Color]::FromArgb(16, 18, 24))

$headerBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(31, 41, 55))
$windowBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(11, 15, 20))
$titleBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(226, 232, 240))
$textBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(203, 213, 225))
$accentBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(34, 197, 94))
$redBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(248, 113, 113))
$yellowBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(250, 204, 21))

$graphics.FillRectangle($windowBrush, 0, 0, $width, $height)
$graphics.FillRectangle($headerBrush, 0, 0, $width, 54)
$graphics.FillEllipse($redBrush, 22, 18, 14, 14)
$graphics.FillEllipse($yellowBrush, 44, 18, 14, 14)
$graphics.FillEllipse($accentBrush, 66, 18, 14, 14)

$graphics.DrawString($Title, $titleFont, $titleBrush, 104, 14)

$y = 78
foreach ($line in $lines[2..($lines.Count - 1)]) {
    $graphics.DrawString($line, $font, $textBrush, $padding, $y)
    $y += $lineHeight
}

$directory = Split-Path -Parent $OutputPath
if (-not (Test-Path $directory)) {
    New-Item -ItemType Directory -Path $directory | Out-Null
}

$bitmap.Save($OutputPath, [System.Drawing.Imaging.ImageFormat]::Png)

$graphics.Dispose()
$bitmap.Dispose()
$font.Dispose()
$titleFont.Dispose()
$headerBrush.Dispose()
$windowBrush.Dispose()
$titleBrush.Dispose()
$textBrush.Dispose()
$accentBrush.Dispose()
$redBrush.Dispose()
$yellowBrush.Dispose()
