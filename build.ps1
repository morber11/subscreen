$distDir = Join-Path $PSScriptRoot "dist"

Write-Host "Running vet..."
go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Error "vet failed"
    exit $LASTEXITCODE
}

Write-Host "Running tests..."
go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Error "Tests failed"
    exit $LASTEXITCODE
}

if (Test-Path $distDir) {
    Get-ChildItem -Path $distDir | Remove-Item -Recurse -Force
} else {
    New-Item -ItemType Directory -Path $distDir | Out-Null
}

$output = Join-Path $distDir "subscreen.exe"
go build -o $output .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Built: $output"
} else {
    Write-Error "Build failed"
    exit $LASTEXITCODE
}

$hash = Get-FileHash -Path $output -Algorithm SHA256
$shaPath = "$output.sha256"
"$($hash.Hash)  subscreen.exe" | Out-File -FilePath $shaPath -Encoding ASCII

$zipPath = Join-Path $distDir "subscreen.zip"
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
Compress-Archive -Path $output, $shaPath -DestinationPath $zipPath
Write-Host "Zipped: $zipPath"
