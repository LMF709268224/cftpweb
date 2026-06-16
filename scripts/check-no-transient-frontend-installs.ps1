$ErrorActionPreference = "Stop"

$matches = Get-ChildItem -Path . -Recurse -File |
  Where-Object {
    $_.FullName -notmatch '\\node_modules\\' -and
    $_.FullName -notmatch '\\vendor\\' -and
    $_.FullName -notmatch '\\docs\\' -and
    $_.FullName -notmatch '\\ztest\\' -and
    ($_.Name -eq "Dockerfile" -or $_.Extension -in @(".sh", ".ps1", ".yml", ".yaml"))
  } |
  Select-String -Pattern "npm\s+install\s+--no-save|yarn\s+add\s+[^&|;]*--no-lockfile" -CaseSensitive:$false

if ($matches) {
  Write-Host "Transient frontend dependency installs are not allowed. Add dependencies to package.json and lockfile instead." -ForegroundColor Red
  $matches | ForEach-Object {
    Write-Host "$($_.Path):$($_.LineNumber): $($_.Line.Trim())" -ForegroundColor Red
  }
  exit 1
}

Write-Host "OK: no transient frontend dependency installs found."

$candidatePackagePath = "candidateserver/vue-web/package.json"
$candidateDockerfilePath = "candidateserver/Dockerfile"

if (Test-Path $candidatePackagePath) {
  $packageJson = Get-Content -Raw -Path $candidatePackagePath | ConvertFrom-Json
  if (-not $packageJson.dependencies.'@embedpdf/snippet') {
    Write-Host "Missing @embedpdf/snippet in candidateserver/vue-web/package.json dependencies. EmbedPDF imports it at runtime, so it must be locked as a direct dependency." -ForegroundColor Red
    exit 1
  }
}

if (Test-Path $candidateDockerfilePath) {
  $dockerfile = Get-Content -Raw -Path $candidateDockerfilePath
  if ($dockerfile -notmatch "@embedpdf/snippet") {
    Write-Host "Missing @embedpdf/snippet in candidateserver/Dockerfile dependency resolve check." -ForegroundColor Red
    exit 1
  }
}

Write-Host "OK: known frontend runtime dependencies are pinned and checked."
