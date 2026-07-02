#!/bin/bash
set -e

echo "=== PerfInsight Demo ==="
echo ""

# Clean state
echo "[1/5] Cleaning database..."
docker exec perfinsight-db psql -U user -d perfinsight \
  -c "DELETE FROM queries; DELETE FROM traces; DELETE FROM metrics;" \
  > /dev/null
echo "  Done."

# Start collector
echo "[2/5] Starting collector..."
cd /c/Users/dawit/Desktop/perfinsight
go run cmd/collector/main.go &
COLLECTOR_PID=$!
sleep 3
curl -s http://localhost:9000/health > /dev/null && echo "  Collector healthy."

# Start testapp
echo "[3/5] Starting test application..."
go run testapp/main.go &
TESTAPP_PID=$!
sleep 3
echo "  Test app running."

# Generate traffic
echo "[4/5] Generating traffic..."
for i in {1..10}; do
  curl -s http://localhost:8080/orders > /dev/null
done
for i in {1..10}; do
  curl -s http://localhost:8080/fast > /dev/null
done
echo "  Sent 20 requests. Waiting for flush..."
sleep 6

# Run analysis
echo "[5/5] Running analysis..."
echo ""
go run cmd/analyze/main.go -endpoint all

# Cleanup
kill $TESTAPP_PID $COLLECTOR_PID 2>/dev/null || true
echo ""
echo "=== Demo Complete ==="