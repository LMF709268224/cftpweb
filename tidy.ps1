# Get all subdirectories containing a go.mod file
$goModules = Get-ChildItem -Recurse -Filter "go.mod"

foreach ($module in $goModules) {
    # Get the directory path where go.mod is located
    $dir = $module.DirectoryName
    
    Write-Host "----------------------------------------" -ForegroundColor Cyan
    Write-Host "Entering directory: ${dir}" -ForegroundColor Yellow
    
    # Change to the subdirectory and run go mod tidy
    Push-Location $dir
    try {
        go mod tidy
        Write-Host "Successfully executed go mod tidy" -ForegroundColor Green
    }
    catch {
        Write-Warning "Execution failed in directory ${dir}: $_"
    }
    # Restore the previous directory
    Pop-Location
}

Write-Host "----------------------------------------" -ForegroundColor Cyan
Write-Host "All directories processed!" -ForegroundColor Green