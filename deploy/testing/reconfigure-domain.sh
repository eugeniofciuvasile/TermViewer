#!/bin/bash

# Usage: ./reconfigure-domain.sh termviewer.local
NEW_DOMAIN=$1

if [ -z "$NEW_DOMAIN" ]; then
    echo "Usage: ./reconfigure-domain.sh <your-domain.local>"
    exit 1
fi

SSO_DOMAIN="sso.$NEW_DOMAIN"
LOGS_DOMAIN="logs.$NEW_DOMAIN"

echo "==========================================="
echo "  TermViewer Testing Domain Reconfigurator"
echo "==========================================="
echo "New Target Domain: $NEW_DOMAIN"

# 1. Update .env file
if [ -f .env ]; then
    sed -i "s/^TESTING_DOMAIN=.*/TESTING_DOMAIN=$NEW_DOMAIN/" .env
    sed -i "s/^TESTING_SSO_DOMAIN=.*/TESTING_SSO_DOMAIN=$SSO_DOMAIN/" .env
    sed -i "s/^TESTING_LOGS_DOMAIN=.*/TESTING_LOGS_DOMAIN=$LOGS_DOMAIN/" .env
    echo "✓ Updated .env file"
else
    echo "✗ .env file not found!"
    exit 1
fi

# 2. Generate New Multi-Domain Certificate
echo "Generating SSL Certificate for $NEW_DOMAIN, $SSO_DOMAIN, and localhost..."

CERT_DIR="./traefik/certs"
mkdir -p "$CERT_DIR"

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout "$CERT_DIR/server.key" \
  -out "$CERT_DIR/server.crt" \
  -subj "/C=US/ST=State/L=City/O=TermViewer/CN=$NEW_DOMAIN" \
  -addext "subjectAltName = DNS:$NEW_DOMAIN, DNS:$SSO_DOMAIN, DNS:$LOGS_DOMAIN, DNS:localhost, IP:127.0.0.1"

echo "✓ New Certificate Generated"

# 3. Final steps
echo "-------------------------------------------"
echo "Setup complete! Please run these final steps:"
echo "1. Add to /etc/hosts: 127.0.0.1 $NEW_DOMAIN $SSO_DOMAIN $LOGS_DOMAIN"
echo "2. Run: docker compose up -d --build --force-recreate"
echo "==========================================="
