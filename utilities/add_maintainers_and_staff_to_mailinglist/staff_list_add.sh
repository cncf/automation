#!/bin/bash
set -euo pipefail

# --- Configuration ---
# Check if config.txt exists and read the configuration
if [ ! -f "config.txt" ]; then
    echo "Error: config.txt file not found in current directory"
    exit 1
fi

# Safely parse config.txt file instead of sourcing it to prevent shell injection
# Extract TOKEN value
TOKEN=$(grep -E '^TOKEN=' config.txt | cut -d'=' -f2- | head -n1)
if [ -z "${TOKEN:-}" ]; then
    echo "Error: TOKEN variable not found in config.txt file"
    exit 1
fi

# Extract SUBGROUP_ID value and validate it's numeric
SUBGROUP_ID=$(grep -E '^SUBGROUP_ID=' config.txt | cut -d'=' -f2- | head -n1)
if [ -z "${SUBGROUP_ID:-}" ]; then
    echo "Error: SUBGROUP_ID variable not found in config.txt file"
    exit 1
fi

# Validate that SUBGROUP_ID is numeric to prevent injection attacks
if ! [[ "$SUBGROUP_ID" =~ ^[0-9]+$ ]]; then
    echo "Error: SUBGROUP_ID must be numeric only. Got: '$SUBGROUP_ID'"
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

# --- Verbose mode (set VERBOSE=1 to log full email addresses) ---
VERBOSE="${VERBOSE:-0}"

# --- Function to redact email for logging ---
redact_email() {
  local email="$1"
  if [[ "$VERBOSE" == "1" ]]; then
    echo "$email"
  else
    # Extract local and domain parts
    local local_part="${email%%@*}"
    local domain="${email##*@}"
    # Show first 2 chars of local part + *** + @domain
    if [[ ${#local_part} -le 2 ]]; then
      echo "***@$domain"
    else
      echo "${local_part:0:2}***@$domain"
    fi
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
    echo "Error: failed to generate JSON payload for email '$email'" >&2
    return 1
  fi

  local redacted_email
  redacted_email=$(redact_email "$email")
  echo "Adding member: $redacted_email with role: $COMMON_ROLE and delivery: $COMMON_DELIVERY_MODE"

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
  while IFS= read -r email; do
    if [[ -n "$email" ]]; then # Skip empty lines
      add_member "$email"
      echo "" # Add a newline for better readability
    fi
  done < "$EMAIL_FILE"
else
  echo "Error: File '$EMAIL_FILE' not found."
  exit 1
fi

echo "Done adding members."