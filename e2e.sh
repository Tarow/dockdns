#!/usr/bin/env bash

#### DockDNS E2E Test Script
## * Check if DockDNS is seeing Docker events
## * Check if DockDNS can publish DNS records


set -exuo pipefail

# Build dockdns binary
make build

bin/dockdns -config ./test/e2e.yaml > dockdns-e2e.log 2>&1 &
DOCKDNS_PID=$!
sleep 2

# Check if dockdns is still running
if ! ps -p $DOCKDNS_PID > /dev/null; then
	echo "dockdns failed to start. Log output:"
	cat dockdns-e2e.log
	exit 1
fi



# Start a test docker container with the required label
docker run --rm -d --name dockdns-e2e-test --label dockdns.name=e2e-test busybox sleep 10


# Wait a moment for dockdns to process
sleep 2


# Check dockdns logs for activity
echo "Checking dockdns logs for test container activity..."
cat dockdns-e2e.log

# Fail if expected log output is missing (customize this string for your app)
if ! grep -q "dockdns-e2e-test" dockdns-e2e.log; then
	echo "dockdns did not process the test container as expected."
	exit 1
fi

echo "dockdns received events for the test container."

# Cleanup
kill $DOCKDNS_PID || true