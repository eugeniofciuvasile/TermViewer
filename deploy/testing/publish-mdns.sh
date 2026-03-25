#!/bin/bash

# Load environment variables from .env
if [ -f .env ]; then
    export $(grep -v '^#' .env | grep -v '^$' | xargs)
fi

# Detect Real LAN IP
LOCAL_IP=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K\S+')
if [ -z "$LOCAL_IP" ]; then
    LOCAL_IP=$(hostname -I | awk '{print $1}')
fi

DOMAIN=${TESTING_DOMAIN:-termviewer.local}
SSO_DOMAIN=${TESTING_SSO_DOMAIN:-sso.termviewer.local}
LOGS_DOMAIN=${TESTING_LOGS_DOMAIN:-logs.termviewer.local}

if ! command -v avahi-publish &> /dev/null; then
    echo "Error: avahi-utils is not installed."
    exit 1
fi

echo "==========================================="
echo "  TermViewer mDNS Automation (Robust Watchdog)"
echo "==========================================="
echo "Real LAN IP: $LOCAL_IP"
echo "Domains: $DOMAIN, $SSO_DOMAIN, $LOGS_DOMAIN"
echo "==========================================="

PIDS=()

cleanup() {
    echo "Shutting down mDNS..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null
    done
    exit
}

trap cleanup INT TERM EXIT

while true; do
    PIDS=()
    
    avahi-publish -a "$DOMAIN" -R "$LOCAL_IP" &
    PIDS+=($!)
    
    avahi-publish -a "$SSO_DOMAIN" -R "$LOCAL_IP" &
    PIDS+=($!)
    
    avahi-publish -a "$LOGS_DOMAIN" -R "$LOCAL_IP" &
    PIDS+=($!)
    
    echo " mDNS Broadcast Active. Press Ctrl+C to stop."
    
    # Wait for any of the processes to exit
    wait -n
    
    echo " Warning: One or more mDNS processes exited. Restarting in 5s..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null
    done
    sleep 5
done
