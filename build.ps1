$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') {
    throw 'Build this application on Windows.'
}

$buildEnvironment = @{}
foreach ($name in @('GOOS', 'GOARCH', 'CGO_ENABLED')) {
    $buildEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}

Push-Location $PSScriptRoot
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'

    & go run github.com/akavel/rsrc@v0.10.2 -manifest app.manifest -o resource_windows_amd64.syso -arch amd64
    if ($LASTEXITCODE -ne 0) {
        throw 'Windows resource compilation failed.'
    }

    New-Item -ItemType Directory -Path dist -Force | Out-Null

    & go build -trimpath -ldflags '-H=windowsgui -s -w' -o dist/simple-audio-merger.exe .
    if ($LASTEXITCODE -ne 0) {
        throw 'GUI build failed.'
    }

    & go build -trimpath -ldflags '-s -w' -o dist/simple-audio-merger-cli.exe .
    if ($LASTEXITCODE -ne 0) {
        throw 'CLI build failed.'
    }

    Copy-Item -LiteralPath README.md -Destination dist/README.md -Force
    $archivePath = 'dist/simple-audio-merger-windows-amd64.zip'
    Compress-Archive -LiteralPath dist/simple-audio-merger.exe, dist/simple-audio-merger-cli.exe, dist/README.md -DestinationPath $archivePath -Force
    $checksum = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath dist/SHA256SUMS -Value "$checksum  simple-audio-merger-windows-amd64.zip" -Encoding ascii

    Write-Host "Build complete: $archivePath"
}
finally {
    foreach ($name in $buildEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $buildEnvironment[$name], 'Process')
    }
    Pop-Location
}
