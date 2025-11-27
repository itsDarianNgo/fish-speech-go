#!/bin/bash
set -e

echo "=== Fish-Speech-Go Integration Tests ==="

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Docker is required for integration tests"
    exit 1
fi

# Start services
echo "Starting services..."
cd docker
docker compose -f docker-compose.yml up -d --build

# Wait for services to be healthy
echo "Waiting for services to be ready..."
sleep 10

MAX_RETRIES=60
RETRY_COUNT=0

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -s http://localhost:8080/v1/health | grep -q '"status":"ok"'; then
        echo "Server is ready!"
        break
    fi
    echo "Waiting for server... ($RETRY_COUNT/$MAX_RETRIES)"
    sleep 5
    RETRY_COUNT=$((RETRY_COUNT + 1))
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo "Server failed to start"
    docker compose logs
    docker compose down
    exit 1
fi

# Run integration tests
echo "Running integration tests..."
cd ../go
FISH_SERVER_URL=http://localhost:8080 \
FISH_BACKEND_URL=http://localhost:8081 \
go test -tags=integration -v ./tests/integration/...

TEST_EXIT_CODE=$?

# Cleanup
echo "Cleaning up..."
cd ../docker
docker compose down

exit $TEST_EXIT_CODE
