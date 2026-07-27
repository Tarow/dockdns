#!/usr/bin/env bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    
    # Stop dockdns if running
    if [ -n "${DOCKDNS_PID:-}" ] && kill -0 "$DOCKDNS_PID" 2>/dev/null; then
        kill "$DOCKDNS_PID" || true
    fi
    
    # Remove test containers
    docker rm -f dockdns-e2e-test dockdns-e2e-labeled 2>/dev/null || true
    
    # Stop Technitium
    if [ "${TECHNITIUM_STARTED:-false}" = "true" ]; then
        log_info "Stopping Technitium DNS..."
        docker rm -f technitium-e2e 2>/dev/null || true
    fi
    
    # Clean up temp files
    rm -f dockdns-e2e.log e2e-config.yaml 2>/dev/null || true
}
trap cleanup EXIT

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$PROJECT_ROOT"

# ============================================================================
# Configuration via Environment Variables
# ============================================================================
# 
# The e2e tests can run in two modes:
#
# 1. LOCAL MODE (default): Spins up a local Technitium DNS server
#    - No environment variables needed
#    - Uses docker to run technitium/dns-server
#
# 2. EXTERNAL MODE: Uses external DNS providers
#    - Set E2E_USE_EXTERNAL=true
#    - For Cloudflare: Set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ZONE_ID
#    - For Technitium: Set TECHNITIUM_API_URL and TECHNITIUM_API_TOKEN
#
# Environment Variables:
#   E2E_USE_EXTERNAL      - Set to "true" to use external providers
#   E2E_TEST_ZONE         - Zone name for testing (default: e2e-test.local)
#   
#   # For Cloudflare (when E2E_USE_EXTERNAL=true)
#   CLOUDFLARE_API_TOKEN  - Cloudflare API token
#   CLOUDFLARE_ZONE_ID    - Cloudflare zone ID
#   CLOUDFLARE_ZONE_NAME  - Cloudflare zone name (e.g., example.com)
#
#   # For Technitium (when E2E_USE_EXTERNAL=true or local mode)
#   TECHNITIUM_API_URL    - Technitium API URL (e.g., http://localhost:5380)
#   TECHNITIUM_API_TOKEN  - Technitium API token
#   TECHNITIUM_USERNAME   - Technitium username (default: admin)
#   TECHNITIUM_PASSWORD   - Technitium password (default: admin123)
#
# ============================================================================

E2E_USE_EXTERNAL="${E2E_USE_EXTERNAL:-false}"
E2E_TEST_ZONE="${E2E_TEST_ZONE:-e2e-test.local}"
TECHNITIUM_USERNAME="${TECHNITIUM_USERNAME:-admin}"
TECHNITIUM_PASSWORD="${TECHNITIUM_PASSWORD:-admin123}"
TECHNITIUM_PORT="${TECHNITIUM_PORT:-5380}"
TECHNITIUM_STARTED="false"

# ============================================================================
# Build dockdns
# ============================================================================
log_info "Building dockdns..."
make build

# ============================================================================
# Start Technitium DNS Server (Local Mode)
# ============================================================================
start_local_technitium() {
    log_info "Starting local Technitium DNS server..."
    
    # Check if port is already in use
    if nc -z localhost "$TECHNITIUM_PORT" 2>/dev/null; then
        log_warn "Port $TECHNITIUM_PORT already in use, attempting to stop existing container..."
        docker rm -f technitium-e2e 2>/dev/null || true
        sleep 2
    fi
    
    docker run -d \
        --name technitium-e2e \
        -p "${TECHNITIUM_PORT}:5380" \
        -e "DNS_SERVER_DOMAIN=dns-server" \
        -e "DNS_SERVER_ADMIN_PASSWORD=${TECHNITIUM_PASSWORD}" \
        technitium/dns-server:latest
    
    TECHNITIUM_STARTED="true"
    
    # Wait for Technitium to be ready
    log_info "Waiting for Technitium to be ready..."
    for i in {1..60}; do
        if curl -s "http://localhost:${TECHNITIUM_PORT}/api/user/login" > /dev/null 2>&1; then
            log_info "Technitium DNS Server is ready!"
            break
        fi
        if [ "$i" -eq 60 ]; then
            log_error "Technitium failed to start within timeout"
            exit 1
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    
    # Additional wait for full initialization
    sleep 3
    
    # Get API token
    log_info "Authenticating with Technitium..."
    local login_response
    login_response=$(curl -s -X POST "http://localhost:${TECHNITIUM_PORT}/api/user/login" \
        -d "user=${TECHNITIUM_USERNAME}&pass=${TECHNITIUM_PASSWORD}")
    
    TECHNITIUM_API_TOKEN=$(echo "$login_response" | jq -r '.token')
    
    if [ -z "$TECHNITIUM_API_TOKEN" ] || [ "$TECHNITIUM_API_TOKEN" = "null" ]; then
        log_error "Failed to get Technitium API token"
        echo "Response: $login_response"
        exit 1
    fi
    
    log_info "Got API token: ${TECHNITIUM_API_TOKEN:0:20}..."
    
    # Create test zone
    log_info "Creating test zone: $E2E_TEST_ZONE"
    curl -s -X POST "http://localhost:${TECHNITIUM_PORT}/api/zones/create" \
        -d "token=${TECHNITIUM_API_TOKEN}&zone=${E2E_TEST_ZONE}&type=Primary" | jq '.'
    
    export TECHNITIUM_API_URL="http://localhost:${TECHNITIUM_PORT}"
    export TECHNITIUM_API_TOKEN
}

# ============================================================================
# Generate e2e config
# ============================================================================
generate_config() {
    log_info "Generating e2e config..."
    
    cat > e2e-config.yaml << EOF
interval: 3
debounceTime: 1
maxDebounceTime: 10

webUI: false

log:
  level: debug
  format: simple

zones:
  - name: ${E2E_TEST_ZONE}
    provider: technitium
    apiURL: ${TECHNITIUM_API_URL}
    apiToken: ${TECHNITIUM_API_TOKEN}

dns:
  a: true
  aaaa: false
  defaultTTL: 300
  purgeUnknown: false

domains:
  - name: "static.${E2E_TEST_ZONE}"
    a: 10.0.0.1
    comment: "Static test record"
EOF

    if [ "$E2E_USE_EXTERNAL" = "true" ] && [ -n "${CLOUDFLARE_API_TOKEN:-}" ]; then
        log_info "Adding Cloudflare zone to config..."
        cat >> e2e-config.yaml << EOF

  - name: ${CLOUDFLARE_ZONE_NAME}
    provider: cloudflare
    apiToken: ${CLOUDFLARE_API_TOKEN}
    zoneID: ${CLOUDFLARE_ZONE_ID}
EOF
    fi
    
    log_info "Config generated:"
    cat e2e-config.yaml
}

# ============================================================================
# Verify DNS records
# ============================================================================
verify_technitium_record() {
    local record_name="$1"
    local expected_value="$2"
    local record_type="${3:-A}"
    local max_attempts="${4:-10}"
    
    log_info "Verifying record: $record_name ($record_type) = $expected_value"
    
    for attempt in $(seq 1 "$max_attempts"); do
        local records
        records=$(curl -s "http://localhost:${TECHNITIUM_PORT}/api/zones/records/get?token=${TECHNITIUM_API_TOKEN}&domain=${record_name}&zone=${E2E_TEST_ZONE}")
        
        local found_value
        found_value=$(echo "$records" | jq -r ".response.records[] | select(.type == \"$record_type\") | .rData.ipAddress // .rData.cname // .rData.value" 2>/dev/null | head -1)
        
        if [ "$found_value" = "$expected_value" ]; then
            log_info "✓ Record verified: $record_name = $found_value (attempt $attempt)"
            return 0
        fi
        
        if [ "$attempt" -lt "$max_attempts" ]; then
            log_warn "Record not yet ready, retrying in 3s... (attempt $attempt/$max_attempts)"
            sleep 3
        fi
    done
    
    log_error "✗ Record mismatch after $max_attempts attempts: expected '$expected_value', got '$found_value'"
    echo "Full response: $records"
    return 1
}

list_all_records() {
    log_info "Listing all records in zone $E2E_TEST_ZONE..."
    curl -s "http://localhost:${TECHNITIUM_PORT}/api/zones/records/get?token=${TECHNITIUM_API_TOKEN}&domain=${E2E_TEST_ZONE}&zone=${E2E_TEST_ZONE}&listZone=true" | \
        jq '.response.records[] | {name: .name, type: .type, rData: .rData}'
}

# ============================================================================
# Main Test Flow
# ============================================================================

# Step 1: Start local Technitium (unless using external)
if [ "$E2E_USE_EXTERNAL" != "true" ]; then
    start_local_technitium
else
    log_info "Using external providers..."
    if [ -z "${TECHNITIUM_API_URL:-}" ] || [ -z "${TECHNITIUM_API_TOKEN:-}" ]; then
        log_error "E2E_USE_EXTERNAL=true but TECHNITIUM_API_URL and TECHNITIUM_API_TOKEN not set"
        exit 1
    fi
fi

# Step 2: Generate config
generate_config

# Step 3: Start dockdns
log_info "Starting dockdns..."
./bin/dockdns -config e2e-config.yaml > dockdns-e2e.log 2>&1 &
DOCKDNS_PID=$!
sleep 3

# Verify dockdns is running
if ! kill -0 "$DOCKDNS_PID" 2>/dev/null; then
    log_error "dockdns failed to start"
    cat dockdns-e2e.log
    exit 1
fi
log_info "dockdns started (PID: $DOCKDNS_PID)"

# Step 4: Verify static record was created
log_info "Waiting for initial sync..."
sleep 5
verify_technitium_record "static.${E2E_TEST_ZONE}" "10.0.0.1" "A"

# Step 5: Start a test container with dockdns labels
log_info "Starting labeled test container..."
docker run -d \
    --name dockdns-e2e-labeled \
    --label "dockdns.name=container.${E2E_TEST_ZONE}" \
    --label "dockdns.a=10.0.0.100" \
    --label "dockdns.comment=E2E test container" \
    busybox sleep 120

# Step 6: Wait for dockdns to detect the container
log_info "Waiting for dockdns to detect container..."
sleep 10

# Step 7: Verify container record was created
verify_technitium_record "container.${E2E_TEST_ZONE}" "10.0.0.100" "A"

# Step 8: Test container removal
log_info "Stopping labeled container..."
docker rm -f dockdns-e2e-labeled

# Wait for dockdns to process the removal
sleep 5

# Step 9: List all records (for debugging)
list_all_records

# Step 10: Show dockdns logs
log_info "dockdns logs:"
cat dockdns-e2e.log

log_info "============================================"
log_info "E2E Tests PASSED!"
log_info "============================================"