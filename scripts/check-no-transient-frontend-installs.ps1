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
