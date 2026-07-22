#!/bin/bash
set -e

echo "
╔══════════════════════════════════════════════╗
║ PerfInsight — Live Demo                     ║
╚══════════════════════════════════════════════╝
"
# Create .env with defaults if not present (for fresh clones)
if [ ! -f .env ]; then
  cat > .env << 'EOF'
POSTGRES_USER=perfinsight
POSTGRES_PASSWORD=perfinsight_secret
POSTGRES_DB=perfinsight
EOF
  echo "[Setup] Created .env with default credentials"
fi

# Stop any existing services
echo "[Setup] Stopping any existing services..."
docker-compose down -v 2>/dev/null || true
pkill -f testapp 2>/dev/null || true
sleep 1

# Start Docker services
echo "[1/5] Starting collector + PostgreSQL via Docker..."
docker-compose up --build -d

# Wait for healthy status
echo "[1/5] Waiting for services to be healthy..."
for i in {1..30}; do
  STATUS=$(docker inspect --format='{{.State.Health.Status}}' perfinsight-collector 2>/dev/null || echo "starting")
  if [ "$STATUS" = "healthy" ]; then
    echo " ✅ Collector is healthy"
    break
  fi
  if [ "$i" = "30" ]; then
    echo " ❌ Collector did not become healthy"
    docker-compose logs collector
    exit 1
  fi
  sleep 2
done

# Start testapp
echo "[2/5] Starting test application..."
go run testapp/main.go &
TESTAPP_PID=$!

# Wait until the server is actually accepting requests
for i in {1..30}; do
    if curl -fs http://localhost:8080/fast >/dev/null 2>&1; then
        echo " ✅ Test app running on :8080"
        break
    fi

    if [ "$i" = "30" ]; then
        echo " ❌ Test application did not start"
        kill "$TESTAPP_PID" 2>/dev/null || true
        exit 1
    fi

    sleep 1
done

# Generate traffic
echo "[3/5] Generating traffic with N+1 query pattern..."
for i in {1..10}; do curl -s http://localhost:8080/orders > /dev/null; done
echo "   Sent 10 requests to /orders (N+1 pattern)..."
for i in {1..10}; do curl -s http://localhost:8080/fast > /dev/null; done
echo "   Sent 10 requests to /fast (clean endpoint)..."
echo "   Waiting for telemetry flush..."
sleep 6

# Run analysis
echo "[4/5] Running performance analysis..."
export DATABASE_URL="host=localhost user=perfinsight password=perfinsight_secret dbname=perfinsight sslmode=disable"
go run cmd/analyze/main.go -endpoint all

# Cleanup
echo "[5/5] Cleaning up..."
kill $TESTAPP_PID 2>/dev/null || true
docker-compose down

echo "
╔══════════════════════════════════════════════╗
║ Demo Complete                               ✅║
╚══════════════════════════════════════════════╝
"