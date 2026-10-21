param([string]$OutputDirectory = 'dist')
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
$previousDirectory = Get-Location
$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$previousCGO = $env:CGO_ENABLED
try {
    Set-Location -LiteralPath $projectRoot
    New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
    $exePath = Join-Path $OutputDirectory 'Portable-Playroom-Windows-amd64.exe'
    $zipPath = Join-Path $OutputDirectory 'Portable-Playroom-Windows-amd64.zip'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go build -trimpath -ldflags '-s -w' -o $exePath .
    if ($LASTEXITCODE -ne 0) { throw 'Falló la compilación.' }
    Copy-Item -LiteralPath 'README.md','THIRD_PARTY_NOTICES.txt' -Destination $OutputDirectory -Force
    Compress-Archive -LiteralPath $exePath,(Join-Path $OutputDirectory 'README.md'),(Join-Path $OutputDirectory 'THIRD_PARTY_NOTICES.txt') -DestinationPath $zipPath -Force
    $checksumLines = @($exePath,$zipPath) | ForEach-Object {
        $checksum = Get-FileHash -Algorithm SHA256 -LiteralPath $_
        $checksum.Hash.ToLowerInvariant() + '  ' + (Split-Path $_ -Leaf)
    }
    [System.IO.File]::WriteAllLines((Join-Path (Join-Path $projectRoot $OutputDirectory) 'SHA256SUMS.txt'), $checksumLines)
    Write-Host "Compilación lista en $OutputDirectory."
} finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
    $env:CGO_ENABLED = $previousCGO
    Set-Location -LiteralPath $previousDirectory
}
