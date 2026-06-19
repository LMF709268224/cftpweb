param(
    [string]$TargetPath
)

$ConfigPath = Join-Path $PSScriptRoot '.golangci.yml'

if ($TargetPath) {
    $FullPath = Resolve-Path $TargetPath -ErrorAction SilentlyContinue
    
    if (-not $FullPath -or -not (Test-Path (Join-Path $FullPath 'go.mod'))) {
        Write-Host ('Error: Directory [{0}] not found or missing go.mod!' -f $TargetPath) -ForegroundColor Red
        exit 1
    }

    Write-Host ('Linting specified module: {0}' -f $FullPath) -ForegroundColor Cyan
    Push-Location $FullPath
    try {
        golangci-lint run --config $ConfigPath ./...
    } finally {
        Pop-Location
    }
} else {
    Get-ChildItem -Recurse -Filter 'go.mod' | ForEach-Object {
        Write-Host ('Linting module: {0}' -f $_.DirectoryName) -ForegroundColor Cyan
        
        Push-Location $_.DirectoryName
        try {
            golangci-lint run --config $ConfigPath ./...
        } finally {
            Pop-Location
        }
    }
}
