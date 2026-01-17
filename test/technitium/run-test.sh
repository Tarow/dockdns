#!/bin/bash
set -e

echo "Starting Technitium DNS Server test..."

cd "$(dirname "$0")"

# Start Technitium DNS Server
echo "Starting Technitium DNS Server..."
docker-compose up -d

# Wait for Technitium to be ready
echo "Waiting for Technitium to be ready..."
for i in {1..60}; do
  if curl -s http://localhost:5380/api/user/login > /dev/null 2>&1; then
    echo "Technitium DNS Server is ready!"
    break
  fi
  echo "Waiting... ($i/60)"
  sleep 2
done

# Additional wait for DNS server to be fully initialized
sleep 5

# Default credentials for Technitium
ADMIN_USER="admin"
ADMIN_PASS="${TECHNITIUM_PASSWORD:-admin123}"

# Create the test zone in Technitium
echo "Creating test zone in Technitium..."
# Login and get a token
TOKEN=$(curl -s -X POST "http://localhost:5380/api/user/login" \
  -d "user=$ADMIN_USER&pass=$ADMIN_PASS" | jq -r '.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Failed to login to Technitium with credentials user=$ADMIN_USER"
  echo "Cleaning up..."
  cd test/technitium
  docker-compose down -v
  exit 1
fi

echo "Successfully logged in. Token: ${TOKEN:0:20}..."

# Create the zone
echo "Creating test.local zone..."
curl -s -X POST "http://localhost:5380/api/zones/create" \
  -d "token=$TOKEN&zone=test.local&type=Primary" | jq '.'

# Run DockDNS with test config
echo "Running DockDNS with Technitium provider..."
cd ../..
./bin/dockdns -config test/technitium/test-config.yaml

# Verify records were created
echo "Verifying records in Technitium..."
RECORDS=$(curl -s "http://localhost:5380/api/zones/records/get?token=$TOKEN&domain=test.local&zone=test.local&listZone=true")
echo "Records in test.local:"
echo "$RECORDS" | jq '.response.records[] | {name: .name, type: .type, rData: .rData}'

# Cleanup
echo "Cleaning up..."
cd test/technitium
docker-compose down -v

echo "Test completed successfully!"
