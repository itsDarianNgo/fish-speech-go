# Fish-Speech-Go Integration Tests (PowerShell)

Write-Host "=== Fish-Speech-Go Integration Tests ===" -ForegroundColor Cyan

# Check if Docker is available
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "Docker is required for integration tests" -ForegroundColor Red
    exit 1
}

# Start services
Write-Host "Starting services..." -ForegroundColor Yellow
Push-Location docker
docker compose -f docker-compose.yml up -d --build

# Wait for services to be ready
Write-Host "Waiting for services to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

$maxRetries = 60
$retryCount = 0

while ($retryCount -lt $maxRetries) {
    try {
        $response = Invoke-RestMethod -Uri "http://localhost:8080/v1/health" -Method Get -ErrorAction Stop
        if ($response.status -eq "ok") {
            Write-Host "Server is ready!" -ForegroundColor Green
            break
        }
    } catch {
        Write-Host "Waiting for server... ($retryCount/$maxRetries)"
        Start-Sleep -Seconds 5
        $retryCount++
    }
}

if ($retryCount -eq $maxRetries) {
    Write-Host "Server failed to start" -ForegroundColor Red
    docker compose logs
    docker compose down
    Pop-Location
    exit 1
}

# Run integration tests
Write-Host "Running integration tests..." -ForegroundColor Yellow
Pop-Location
Push-Location go

$env:FISH_SERVER_URL = "http://localhost:8080"
$env:FISH_BACKEND_URL = "http://localhost:8081"
go test -tags=integration -v ./tests/integration/...

$testExitCode = $LASTEXITCODE

# Cleanup
Write-Host "Cleaning up..." -ForegroundColor Yellow
Pop-Location
Push-Location docker
docker compose down
Pop-Location

exit $testExitCode
