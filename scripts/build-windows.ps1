$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
$previousDirectory = Get-Location
$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$previousCGO = $env:CGO_ENABLED
try {
    Set-Location -LiteralPath $projectRoot
    New-Item -ItemType Directory -Force -Path 'dist' | Out-Null
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go build -trimpath -ldflags '-s -w' -o 'dist/Portable-Playroom-Windows-amd64.exe' .
    if ($LASTEXITCODE -ne 0) { throw 'Falló la compilación.' }
    Copy-Item -LiteralPath 'README.md','THIRD_PARTY_NOTICES.txt' -Destination 'dist' -Force
    Compress-Archive -LiteralPath 'dist/Portable-Playroom-Windows-amd64.exe','dist/README.md','dist/THIRD_PARTY_NOTICES.txt' -DestinationPath 'dist/Portable-Playroom-Windows-amd64.zip' -Force
    $checksumLines = @('dist/Portable-Playroom-Windows-amd64.exe','dist/Portable-Playroom-Windows-amd64.zip') | ForEach-Object {
        $checksum = Get-FileHash -Algorithm SHA256 -LiteralPath $_
        $checksum.Hash.ToLowerInvariant() + '  ' + (Split-Path $_ -Leaf)
    }
    [System.IO.File]::WriteAllLines((Join-Path $projectRoot 'dist/SHA256SUMS.txt'), $checksumLines)
    Write-Host 'Compilación lista en dist/.'
} finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
    $env:CGO_ENABLED = $previousCGO
    Set-Location -LiteralPath $previousDirectory
}
