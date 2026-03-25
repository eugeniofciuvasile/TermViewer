#!/usr/bin/env bash
# =========================================================================
# TermViewer Interactive Setup
# =========================================================================
# Generates a complete .env file for testing or production deployment.
# Usage: ./setup.sh
# =========================================================================

set -euo pipefail

# ── Colours & helpers ────────────────────────────────────────────────────

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

banner() { printf "\n${CYAN}${BOLD}══════════════════════════════════════════${NC}\n  %s\n${CYAN}${BOLD}══════════════════════════════════════════${NC}\n\n" "$1"; }
section() { printf "\n${YELLOW}── %s ──${NC}\n\n" "$1"; }
info()    { printf "${GREEN}✔${NC} %s\n" "$1"; }
warn()    { printf "${YELLOW}⚠${NC} %s\n" "$1"; }
err()     { printf "${RED}✖${NC} %s\n" "$1" >&2; }

# Generate a cryptographically strong random string
gen_secret() {
  local len="${1:-32}"
  openssl rand -base64 "$len" 2>/dev/null | tr -d '/+=' | head -c "$len"
}

# Prompt with a default value. Empty input → default.
ask() {
  local var_name="$1" prompt="$2" default="$3"
  local input
  if [[ -n "$default" ]]; then
    printf "  ${BOLD}%s${NC} [${CYAN}%s${NC}]: " "$prompt" "$default"
  else
    printf "  ${BOLD}%s${NC}: " "$prompt"
  fi
  read -r input
  eval "$var_name=\"${input:-$default}\""
}

# Prompt for a secret – show that one will be generated, allow override.
ask_secret() {
  local var_name="$1" prompt="$2" generated
  generated="$(gen_secret 32)"
  printf "  ${BOLD}%s${NC} [${CYAN}auto-generated${NC}]: " "$prompt"
  read -r input
  eval "$var_name=\"${input:-$generated}\""
}

# Prompt for a password – enforce minimum length in production.
ask_password() {
  local var_name="$1" prompt="$2" min_len="${3:-16}" generated
  generated="$(gen_secret "$min_len")"
  printf "  ${BOLD}%s${NC} [${CYAN}auto-generated${NC}]: " "$prompt"
  read -r input
  local value="${input:-$generated}"
  if [[ "$ENV" == "production" && ${#value} -lt $min_len ]]; then
    warn "Password too short for production (min $min_len chars). Using generated password."
    value="$generated"
  fi
  eval "$var_name=\"$value\""
}

# Yes/No prompt, default Y
confirm() {
  local prompt="$1"
  printf "  ${BOLD}%s${NC} [Y/n]: " "$prompt"
  read -r yn
  [[ -z "$yn" || "$yn" =~ ^[Yy] ]]
}

# ── Environment selection ────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

banner "TermViewer Setup"

printf "Select environment:\n"
printf "  ${BOLD}1)${NC} testing   – self-signed certs, termviewer.local\n"
printf "  ${BOLD}2)${NC} production – Let's Encrypt, your real domain\n\n"
printf "  Choice [1]: "
read -r env_choice

case "${env_choice:-1}" in
  2) ENV="production" ;;
  *) ENV="testing" ;;
esac

TARGET_DIR="$SCRIPT_DIR/$ENV"
ENV_FILE="$TARGET_DIR/.env"

info "Environment: $ENV"

if [[ -f "$ENV_FILE" ]]; then
  warn "An existing .env file was found at $ENV_FILE"
  if ! confirm "Overwrite it?"; then
    err "Aborted."
    exit 1
  fi
fi

# ── Domain ───────────────────────────────────────────────────────────────

section "Domain Configuration"

if [[ "$ENV" == "production" ]]; then
  ask DOMAIN "Your domain (e.g. termviewer.example.com)" ""
  while [[ -z "$DOMAIN" || "$DOMAIN" == "yourdomain.com" ]]; do
    err "A real domain is required for production."
    ask DOMAIN "Your domain" ""
  done
  ask ACME_EMAIL "Email for Let's Encrypt certificates" ""
  while [[ -z "$ACME_EMAIL" || "$ACME_EMAIL" == *"example.com" ]]; do
    err "A real email address is required for Let's Encrypt."
    ask ACME_EMAIL "Email for Let's Encrypt" ""
  done
  SSO_DOMAIN="auth.${DOMAIN}"
  LOGS_DOMAIN="logs.${DOMAIN}"
  PROTO="https"
else
  ask TESTING_DOMAIN "Local domain" "termviewer.local"
  DOMAIN="$TESTING_DOMAIN"
  SSO_DOMAIN="sso.${DOMAIN}"
  LOGS_DOMAIN="logs.${DOMAIN}"
  PROTO="https"
fi

info "Frontend:  ${PROTO}://${DOMAIN}"
info "SSO:       ${PROTO}://${SSO_DOMAIN}"
info "Logs:      ${PROTO}://${LOGS_DOMAIN}"

# ── Database ─────────────────────────────────────────────────────────────

section "Database (PostgreSQL)"

ask POSTGRES_USER       "Database user"           "keycloak"
ask POSTGRES_BOOTSTRAP_DB "Bootstrap database"    "keycloak"
ask POSTGRES_APP_DB     "Application database"    "termviewer"
ask_password POSTGRES_PASSWORD "Database password" 20

info "Database user: $POSTGRES_USER"
info "Password: ${POSTGRES_PASSWORD:0:4}****"

# ── Keycloak ─────────────────────────────────────────────────────────────

section "Keycloak (Identity Provider)"

ask KC_REALM "Keycloak realm" "termviewer"
ask KC_ADMIN_ROLE "Admin role name" "termviewer-admin"

ask KEYCLOAK_ADMIN "Keycloak admin username" "admin"

if [[ "$ENV" == "production" ]]; then
  printf "  Generate a secure Keycloak admin password automatically?\n"
  if confirm "Auto-generate password?"; then
    KEYCLOAK_ADMIN_PASSWORD="$(gen_secret 24)"
    info "Keycloak admin password: ${KEYCLOAK_ADMIN_PASSWORD:0:4}**** (auto-generated)"
  else
    ask_password KEYCLOAK_ADMIN_PASSWORD "Keycloak admin password" 20
    if [[ "$KEYCLOAK_ADMIN_PASSWORD" == "admin" ]]; then
      warn "Refusing 'admin' as production password. Generating a secure one."
      KEYCLOAK_ADMIN_PASSWORD="$(gen_secret 24)"
    fi
  fi
else
  printf "  Generate a secure password, or use a simple default for testing?\n"
  if confirm "Auto-generate password? (No = use 'admin')"; then
    KEYCLOAK_ADMIN_PASSWORD="$(gen_secret 20)"
    info "Keycloak admin password: ${KEYCLOAK_ADMIN_PASSWORD:0:4}**** (auto-generated)"
  else
    KEYCLOAK_ADMIN_PASSWORD="admin"
    info "Keycloak admin password: admin (testing default)"
  fi
fi

info "Keycloak admin: $KEYCLOAK_ADMIN"
info "Keycloak hostname: $SSO_DOMAIN"

# ── Frontend (NextAuth) ─────────────────────────────────────────────────

section "Frontend (NextAuth)"

ask_secret NEXTAUTH_SECRET "NextAuth session secret"

info "NextAuth secret: ${NEXTAUTH_SECRET:0:6}****"

# ── Dozzle (Logs) ───────────────────────────────────────────────────────

section "Dozzle (Log Viewer)"

printf "  Dozzle requires a users.yml file with bcrypt-hashed passwords.\n"
printf "  The setup will generate this file for you.\n\n"

ask DOZZLE_USERNAME "Dozzle admin username" "admin"
ask DOZZLE_EMAIL   "Dozzle admin email (for Gravatar)" "admin@${DOMAIN}"

if [[ "$ENV" == "production" ]]; then
  printf "  Generate a secure Dozzle password automatically?\n"
  if confirm "Auto-generate password?"; then
    DOZZLE_PASSWORD="$(gen_secret 20)"
  else
    ask_password DOZZLE_PASSWORD "Dozzle admin password" 16
    if [[ "$DOZZLE_PASSWORD" == "admin" ]]; then
      warn "Refusing 'admin' as production password. Generating a secure one."
      DOZZLE_PASSWORD="$(gen_secret 20)"
    fi
  fi
else
  if confirm "Auto-generate password? (No = use 'admin')"; then
    DOZZLE_PASSWORD="$(gen_secret 16)"
  else
    DOZZLE_PASSWORD="admin"
    info "Dozzle password: admin (testing default)"
  fi
fi

# Generate bcrypt hash using the dozzle docker image
info "Generating bcrypt hash for Dozzle password..."
DOZZLE_BCRYPT=$(docker run --rm amir20/dozzle:latest generate "$DOZZLE_USERNAME" --password "$DOZZLE_PASSWORD" --email "$DOZZLE_EMAIL" --name "$DOZZLE_USERNAME" 2>/dev/null)

if [[ -z "$DOZZLE_BCRYPT" ]]; then
  # Fallback: generate bcrypt with htpasswd or python if docker fails
  if command -v htpasswd &>/dev/null; then
    DOZZLE_HASH=$(htpasswd -nbBC 10 "" "$DOZZLE_PASSWORD" | tr -d ':\n' | sed 's/^\$//')
    DOZZLE_HASH="\$$DOZZLE_HASH"
  elif command -v python3 &>/dev/null; then
    DOZZLE_HASH=$(python3 -c "import bcrypt; print(bcrypt.hashpw(b'${DOZZLE_PASSWORD}', bcrypt.gensalt(10)).decode())" 2>/dev/null)
  fi

  if [[ -z "$DOZZLE_HASH" ]]; then
    err "Could not generate bcrypt hash. Please install docker, htpasswd, or python3+bcrypt."
    exit 1
  fi

  DOZZLE_BCRYPT="users:
  ${DOZZLE_USERNAME}:
    email: ${DOZZLE_EMAIL}
    name: ${DOZZLE_USERNAME}
    password: ${DOZZLE_HASH}"
fi

# Write users.yml
DOZZLE_DIR="$TARGET_DIR/dozzle"
mkdir -p "$DOZZLE_DIR"
echo "$DOZZLE_BCRYPT" > "$DOZZLE_DIR/users.yml"
chmod 600 "$DOZZLE_DIR/users.yml"

info "Dozzle users.yml written to: $DOZZLE_DIR/users.yml"

# ── SMTP ─────────────────────────────────────────────────────────────────

section "SMTP (Email)"

printf "  Configure email sending now? (needed for user activation emails)\n"
if confirm "Set up SMTP?"; then
  ask SMTP_HOST "SMTP host"               "smtp.resend.com"
  ask SMTP_PORT "SMTP port"               "587"
  ask SMTP_USER "SMTP username"           "resend"
  ask SMTP_PASS "SMTP password / API key" ""
  while [[ -z "$SMTP_PASS" || "$SMTP_PASS" == "your_resend_api_key_here" ]]; do
    err "An SMTP password/API key is required."
    ask SMTP_PASS "SMTP password / API key" ""
  done
  if [[ "$ENV" == "production" ]]; then
    ask SMTP_FROM "Sender address" "hello@${DOMAIN}"
  else
    ask SMTP_FROM "Sender address" "onboarding@resend.dev"
  fi
  info "SMTP configured: $SMTP_HOST:$SMTP_PORT"
else
	SMTP_HOST=""
	SMTP_PORT=""
	SMTP_USER=""
	SMTP_PASS=""
	SMTP_FROM="noreply@${DOMAIN}"
	warn "SMTP skipped – email features will not work until configured."
fi

# ── TLS ──────────────────────────────────────────────────────────────────

section "TLS / Security"

if [[ "$ENV" == "testing" ]]; then
  CERT_FILE="$SCRIPT_DIR/testing/traefik/certs/server.crt"
  if [[ -f "$CERT_FILE" ]]; then
    DETECTED_FP=$(openssl x509 -in "$CERT_FILE" -noout -fingerprint -sha256 2>/dev/null | sed 's/.*=//;s/://g' | tr '[:upper:]' '[:lower:]')
    if [[ -n "$DETECTED_FP" ]]; then
      info "Detected self-signed certificate fingerprint from traefik/certs/server.crt"
      printf "  ${BOLD}SHA-256:${NC} ${CYAN}%s${NC}\n" "$DETECTED_FP"
      if confirm "Use this fingerprint for mobile app TOFU?"; then
        SERVER_TLS_FINGERPRINT="$DETECTED_FP"
      else
        ask SERVER_TLS_FINGERPRINT "TLS fingerprint (or leave empty)" ""
      fi
    else
      warn "Could not read fingerprint from certificate."
      ask SERVER_TLS_FINGERPRINT "TLS fingerprint (optional)" ""
    fi
  else
    printf "  No self-signed certificate found at traefik/certs/server.crt\n"
    printf "  If you use self-signed certificates, paste the SHA-256 fingerprint\n"
    printf "  so the mobile app can perform trust-on-first-use (TOFU).\n"
    ask SERVER_TLS_FINGERPRINT "TLS fingerprint (optional)" ""
  fi
else
  SERVER_TLS_FINGERPRINT=""
  info "Production uses Let's Encrypt – no fingerprint needed."
fi

# ── Write .env ───────────────────────────────────────────────────────────

section "Writing configuration"

if [[ "$ENV" == "testing" ]]; then
  cat > "$ENV_FILE" <<ENVEOF
# =========================================================================
# TermViewer Configuration – Testing
# Generated by setup.sh on $(date -u +"%Y-%m-%d %H:%M:%S UTC")
# =========================================================================

# --- Domain ---
TESTING_DOMAIN=${DOMAIN}
TESTING_SSO_DOMAIN=${SSO_DOMAIN}
TESTING_LOGS_DOMAIN=${LOGS_DOMAIN}

# --- Database (PostgreSQL) ---
POSTGRES_BOOTSTRAP_DB=${POSTGRES_BOOTSTRAP_DB}
POSTGRES_USER=${POSTGRES_USER}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_APP_DB=${POSTGRES_APP_DB}

# --- Keycloak ---
KEYCLOAK_ADMIN=${KEYCLOAK_ADMIN}
KEYCLOAK_ADMIN_PASSWORD=${KEYCLOAK_ADMIN_PASSWORD}
KEYCLOAK_HOSTNAME=${SSO_DOMAIN}

# --- Backend (Go) ---
KC_REALM=${KC_REALM}
KC_ADMIN_ROLE=${KC_ADMIN_ROLE}

# --- Frontend (Next.js) ---
NEXTAUTH_SECRET=${NEXTAUTH_SECRET}
NEXTAUTH_URL=${PROTO}://${DOMAIN}
FRONTEND_BASE_URL=${PROTO}://${DOMAIN}
NEXT_PUBLIC_API_URL=${PROTO}://${DOMAIN}

# --- TLS Security ---
SERVER_TLS_FINGERPRINT=${SERVER_TLS_FINGERPRINT}

# --- SMTP (Email) ---
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASS=${SMTP_PASS}
SMTP_FROM=${SMTP_FROM}
ENVEOF

else
  # ── Production ──
  cat > "$ENV_FILE" <<ENVEOF
# =========================================================================
# TermViewer Configuration – Production
# Generated by setup.sh on $(date -u +"%Y-%m-%d %H:%M:%S UTC")
# =========================================================================

# --- Domain ---
DOMAIN=${DOMAIN}
ACME_EMAIL=${ACME_EMAIL}

# --- Database (PostgreSQL) ---
POSTGRES_BOOTSTRAP_DB=${POSTGRES_BOOTSTRAP_DB}
POSTGRES_USER=${POSTGRES_USER}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_APP_DB=${POSTGRES_APP_DB}

# --- Keycloak ---
KEYCLOAK_ADMIN=${KEYCLOAK_ADMIN}
KEYCLOAK_ADMIN_PASSWORD=${KEYCLOAK_ADMIN_PASSWORD}
KEYCLOAK_HOSTNAME=auth.${DOMAIN}

# --- Backend (Go) ---
KC_REALM=${KC_REALM}
KC_ADMIN_ROLE=${KC_ADMIN_ROLE}

# --- Frontend (Next.js) ---
NEXTAUTH_SECRET=${NEXTAUTH_SECRET}
NEXTAUTH_URL=https://${DOMAIN}
FRONTEND_BASE_URL=https://${DOMAIN}
NEXT_PUBLIC_API_URL=https://${DOMAIN}

# --- SMTP (Email) ---
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASS=${SMTP_PASS}
SMTP_FROM=${SMTP_FROM}
ENVEOF
fi

chmod 600 "$ENV_FILE"
info "Configuration written to: $ENV_FILE"
info "File permissions set to 600 (owner-only read/write)"

# ── Summary ──────────────────────────────────────────────────────────────

banner "Setup Complete"

printf "  ${BOLD}Environment:${NC}  %s\n" "$ENV"
printf "  ${BOLD}Config file:${NC}  %s\n" "$ENV_FILE"
printf "\n"
printf "  ${BOLD}Next steps:${NC}\n"

if [[ "$ENV" == "testing" ]]; then
  printf "    1. Add '127.0.0.1 %s %s %s' to /etc/hosts\n" "$DOMAIN" "$SSO_DOMAIN" "$LOGS_DOMAIN"
  printf "    2. cd %s && docker compose up -d --build\n" "$TARGET_DIR"
  printf "    3. Open https://%s in your browser\n" "$DOMAIN"
else
  printf "    1. Point DNS A records for %s, auth.%s, logs.%s to your server\n" "$DOMAIN" "$DOMAIN" "$DOMAIN"
  printf "    2. cd %s && docker compose up -d --build\n" "$TARGET_DIR"
  printf "    3. Open https://%s in your browser\n" "$DOMAIN"
fi

printf "\n"
printf "  ${BOLD}Dozzle (Log Viewer):${NC}\n"
printf "    User:     %s\n" "$DOZZLE_USERNAME"
printf "    Password: %s\n" "$DOZZLE_PASSWORD"
printf "    Config:   %s/dozzle/users.yml\n" "$TARGET_DIR"
printf "\n"
printf "  ${BOLD}Keycloak:${NC}\n"
printf "    User:     %s\n" "$KEYCLOAK_ADMIN"
printf "    Password: %s\n" "$KEYCLOAK_ADMIN_PASSWORD"
printf "\n"
printf "  ${BOLD}Database:${NC}\n"
printf "    User:     %s\n" "$POSTGRES_USER"
printf "    Password: %s\n" "$POSTGRES_PASSWORD"
printf "\n"
warn "Save these credentials now – they won't be shown again!"
info "Secrets are stored in $ENV_FILE – keep this file safe!"

if [[ "$ENV" == "production" ]]; then
  printf "\n"
  warn "Production security reminders:"
  printf "    • Back up your .env file securely – it contains all secrets\n"
  printf "    • Never commit .env files to version control\n"
  printf "    • Rotate passwords and secrets periodically\n"
  printf "    • Ensure your Keycloak realm is configured with proper policies\n"
  printf "    • Monitor logs at https://logs.%s\n" "$DOMAIN"
fi

printf "\n"
