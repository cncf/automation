#!/bin/bash
set -euo pipefail

# --- Configuration ---
# Check if config.txt exists and read the configuration
if [ ! -f "config.txt" ]; then
    echo "Error: config.txt file not found in current directory"
    exit 1
fi

# Source the config.txt file to load variables
source config.txt

# Check if required variables are set
if [ -z "${TOKEN:-}" ]; then
    echo "Error: TOKEN variable not found in config.txt file"
    exit 1
fi

if [ -z "${SUBGROUP_ID:-}" ]; then
    echo "Error: SUBGROUP_ID variable not found in config.txt file"
    exit 1
fi

# Set variables from config.txt
AUTH_TOKEN="$TOKEN"
API_URL="https://api-gw.platform.linuxfoundation.org/project-infrastructure-service/v2/groupsio_subgroup/$SUBGROUP_ID/members"

# --- List of Email Addresses (one per line) ---
EMAIL_FILE="${EMAIL_FILE:-staff_emails.txt}"  # One email per line

# --- Common Role and Delivery Mode ---
COMMON_ROLE="owner"
COMMON_DELIVERY_MODE="email_delivery_single"
COMMON_MEMBER_TYPE="direct"

# --- Logging control ---
# Set VERBOSE=true to log full email addresses (for local debugging)
# Set VERBOSE=false or leave unset to redact emails in logs (recommended for CI/CD)
VERBOSE="${VERBOSE:-false}"

# --- Function to redact email for logging ---
redact_email() {
  local email="$1"
  if [[ "$VERBOSE" == "true" ]]; then
    echo "$email"
  else
    # Redact the local part, show only domain: user@example.com -> ***@example.com
    echo "${email}" | sed -E 's/^[^@]+/***/'
  fi
}

# --- Function to add a member ---
add_member() {
  local email="$1"
  local payload
  if ! payload=$(jq -n \
    --arg email "$email" \
    --arg mod_status "$COMMON_ROLE" \
    --arg delivery_mode "$COMMON_DELIVERY_MODE" \
    --arg member_type "$COMMON_MEMBER_TYPE" \
    '{email: $email, mod_status: $mod_status, delivery_mode: $delivery_mode, member_type: $member_type}'); then
    echo "Error: failed to generate JSON payload for email '$(redact_email "$email")'" >&2
    return 1
  fi

  echo "Adding member: $(redact_email "$email") with role: $COMMON_ROLE and delivery: $COMMON_DELIVERY_MODE"

  local response status body
  response="$(curl -sS -w '\n%{http_code}' -X POST \
    -H "Authorization: Bearer $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payload" \
    "$API_URL")"
  status="$(printf '%s' "$response" | tail -n 1)"
  body="$(printf '%s' "$response" | sed '$d')"

  if [[ "$status" -ge 400 ]]; then
    echo "  API Error (HTTP $status): $body" >&2
    return 1
  fi

  echo "  API Response (HTTP $status): $body"
}

# --- Main execution ---
echo "Attempting to add members from '$EMAIL_FILE' to subgroup ID $SUBGROUP_ID..."

if [ -f "$EMAIL_FILE" ]; then
  member_count=0
  success_count=0
  fail_count=0
  
  while IFS= read -r email; do
    if [[ -n "$email" ]]; then # Skip empty lines
      member_count=$((member_count + 1))
      if add_member "$email"; then
        success_count=$((success_count + 1))
      else
        fail_count=$((fail_count + 1))
      fi
      echo "" # Add a newline for better readability
    fi
  done < "$EMAIL_FILE"
  
  echo "Summary: Processed $member_count member(s) - $success_count succeeded, $fail_count failed."
else
  echo "Error: File '$EMAIL_FILE' not found."
  exit 1
fi

echo "Done adding members."