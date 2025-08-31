#!/bin/bash

# WindZ Monitor Deployment Script
# Deploys the ARM Linux binary to target server

set -e  # Exit on any error

# Load deployment configuration
DEPLOY_ENV_FILE="deploy.env"

if [ ! -f "$DEPLOY_ENV_FILE" ]; then
    echo "Error: $DEPLOY_ENV_FILE not found!"
    echo "Please copy deploy.env.example to deploy.env and configure it."
    echo ""
    echo "Example:"
    echo "  cp deploy.env.example deploy.env"
    echo "  # Edit deploy.env with your server details"
    exit 1
fi

# Source the environment file
source "$DEPLOY_ENV_FILE"

# Configuration with defaults
HOST="${DEPLOY_HOST}"
USER="${DEPLOY_USER}"
BINARY="${DEPLOY_BINARY:-windz}"
SERVICE_NAME="${DEPLOY_SERVICE_NAME}"
TARGET_PATH="${DEPLOY_TARGET_PATH}"
SERVICE_USER="${DEPLOY_SERVICE_USER}"
SSH_PORT="${DEPLOY_SSH_PORT:-22}"
HEALTH_URL="${DEPLOY_HEALTH_URL:-http://localhost:8080/health}"

# Validate required configuration
if [ -z "$HOST" ] || [ -z "$USER" ] || [ -z "$SERVICE_NAME" ] || [ -z "$TARGET_PATH" ] || [ -z "$SERVICE_USER" ]; then
    echo "Error: Missing required configuration in $DEPLOY_ENV_FILE"
    echo "Required variables: DEPLOY_HOST, DEPLOY_USER, DEPLOY_SERVICE_NAME, DEPLOY_TARGET_PATH, DEPLOY_SERVICE_USER"
    exit 1
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    error "Binary '$BINARY' not found. Run 'make linux-arm64' first."
    exit 1
fi

# Check if binary is ARM64 Linux
ARCH=$(file "$BINARY" | grep -o "ARM aarch64" || true)
if [ -z "$ARCH" ]; then
    error "Binary '$BINARY' is not ARM64 Linux. Run 'make linux-arm64' first."
    exit 1
fi

log "Starting deployment of WindZ Monitor to $HOST"
log "Binary: $BINARY ($(ls -lh "$BINARY" | awk '{print $5}'))"

# Step 1: Copy binary to remote server
log "Step 1: Copying binary to $HOST:"
SCP_CMD="scp"
if [ "$SSH_PORT" != "22" ]; then
    SCP_CMD="scp -P $SSH_PORT"
fi

if $SCP_CMD "$BINARY" "$USER@$HOST:"; then
    success "Binary uploaded successfully"
else
    error "Failed to upload binary"
    exit 1
fi

# Step 2: Execute deployment commands on remote server
log "Step 2: Executing deployment commands on remote server"
SSH_CMD="ssh -t"
if [ "$SSH_PORT" != "22" ]; then
    SSH_CMD="ssh -t -p $SSH_PORT"
fi

$SSH_CMD "$USER@$HOST" << EOF
set -e

echo "Stopping service: $SERVICE_NAME"
if sudo /bin/systemctl stop $SERVICE_NAME; then
    echo "✓ Service stopped"
else
    echo "⚠ Failed to stop service (may not be running)"
fi

echo "Backing up current binary..."
if [ -f $TARGET_PATH ]; then
    sudo /bin/cp $TARGET_PATH $TARGET_PATH.backup.\$(date +%Y%m%d-%H%M%S)
    echo "✓ Current binary backed up"
fi

echo "Installing new binary..."
sudo /bin/cp $BINARY $TARGET_PATH
echo "✓ Binary copied to $TARGET_PATH"

echo "Setting ownership..."
sudo /bin/chown $SERVICE_USER:$SERVICE_USER $TARGET_PATH
echo "✓ Ownership set to $SERVICE_USER:$SERVICE_USER"

echo "Setting executable permissions..."
sudo /bin/chmod +x $TARGET_PATH
echo "✓ Permissions set"

echo "Starting service: $SERVICE_NAME"
if sudo /bin/systemctl start $SERVICE_NAME; then
    echo "✓ Service started"
else
    echo "✗ Failed to start service"
    exit 1
fi

echo "Cleaning up temporary file..."
rm -f $BINARY
echo "✓ Cleanup complete"
EOF

if [ $? -eq 0 ]; then
    success "Remote deployment commands completed successfully"
else
    error "Remote deployment failed"
    exit 1
fi

# Step 3: Verify service is running
log "Step 3: Verifying service status"
sleep 3 # Give service time to start

SERVICE_STATUS=$($SSH_CMD "$USER@$HOST" "sudo /bin/systemctl is-active $SERVICE_NAME" 2>/dev/null || echo "failed")

# printf 'DEBUG: "%s"\n' "$SERVICE_STATUS" | od -c
# returns a extra carriage retur, strip it here if one is there
SERVICE_STATUS=$(echo "$SERVICE_STATUS" | tr -d '\r\n')

if [[ "$SERVICE_STATUS" = "active" ]]; then
    success "Service is running"
else
    error "Service is not active (status: $SERVICE_STATUS)"
    
    # Get service logs for debugging
    warn "Getting recent service logs:"
    $SSH_CMD "$USER@$HOST" "sudo /bin/journalctl -u $SERVICE_NAME --no-pager -n 10" || true
    exit 1
fi

# Step 4: Optional health check (if service has health endpoint)
log "Step 4: Performing health check"
sleep 2  # Give service more time to initialize

HEALTH_CHECK=$($SSH_CMD "$USER@$HOST" "curl -s -o /dev/null -w '%{http_code}' '$HEALTH_URL' 2>/dev/null || echo 'failed'")

if [ "$HEALTH_CHECK" = "200" ]; then
    success "Health check passed (HTTP 200)"
elif [ "$HEALTH_CHECK" = "failed" ]; then
    warn "Health check failed (curl error - service may still be starting)"
else
    warn "Health check returned HTTP $HEALTH_CHECK"
fi

# Final status
log "Deployment Summary:"
echo "  • Binary deployed: $(file "$BINARY" | cut -d: -f2 | cut -d, -f1-2)"
echo "  • Service status: $SERVICE_STATUS"
echo "  • Health check: $HEALTH_CHECK ($HEALTH_URL)"

success "Deployment completed successfully!"
log "WindZ Monitor is now running on $HOST"
