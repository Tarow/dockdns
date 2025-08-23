#!/usr/bin/env bash

set -exuo pipefail

# Build dockdns binary
make build


# Start dockdns in the background and capture logs
bin/dockdns > dockdns-e2e.log 2>&1 &
DOCKDNS_PID=$!
sleep 2


# Start a test docker container
docker run --rm -d --name dockdns-e2e-test busybox sleep 10


# Wait a moment for dockdns to process
sleep 2


# Check dockdns logs for activity
echo "Checking dockdns logs for test container activity..."
ps -p $DOCKDNS_PID
cat dockdns-e2e.log


# Cleanup
kill $DOCKDNS_PID || true