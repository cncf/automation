#!/bin/bash
set -euo pipefail

# --- Configuration ---
# Check if config.txt exists and read the configuration
if [ ! -f "config.txt" ]; then
    echo "Error: config.txt file not found in current directory"
    exit 1
fi

# Function to trim leading and trailing whitespace
trim_whitespace() {
    local var="$1"
    # Remove leading whitespace
    var="${var#"${var%%[![:space:]]*}"}"
    # Remove trailing whitespace
    var="${var%"${var##*[![:space:]]}"}"
    echo "$var"
}

# Parse config.txt safely without executing it as shell code
# This prevents command injection via shell metacharacters in the config values
TOKEN=""
SUBGROUP_ID=""

while IFS= read -r line; do
    # Skip empty lines and comments
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    
    # Split on first '=' only to preserve '=' characters in values
    if [[ "$line" =~ ^([^=]+)=(.*)$ ]]; then
        key="${BASH_REMATCH[1]}"
        value="${BASH_REMATCH[2]}"
        
        # Trim whitespace
        key="$(trim_whitespace "$key")"
        value="$(trim_whitespace "$value")"
        
        # Assign to variables based on key
        case "$key" in
            TOKEN)
                TOKEN="$value"
                ;;
            SUBGROUP_ID)
                SUBGROUP_ID="$value"
                ;;
        esac
    fi
done < config.txt

# Check if required variables are set
if [ -z "$TOKEN" ]; then
    echo "Error: TOKEN variable not found in config.txt file"
    exit 1
fi

if [ -z "$SUBGROUP_ID" ]; then
    echo "Error: SUBGROUP_ID variable not found in config.txt file"
    exit 1
fi

# Set variables from config.txt
AUTH_TOKEN="$TOKEN"
API_URL="https://api-gw.platform.linuxfoundation.org/project-infrastructure-service/v2/groupsio_subgroup/$SUBGROUP_ID/members"

# --- List of Email Addresses (one per line) ---
EMAIL_FILE="${EMAIL_FILE:-maintainers_emails.txt}"  # One email per line

# --- Common Role and Delivery Mode ---
COMMON_ROLE="none"
COMMON_DELIVERY_MODE="email_delivery_single"
COMMON_MEMBER_TYPE="direct"

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
    echo "Error: Failed to construct JSON payload for email '$email'." >&2
    exit 1
  fi

  echo "Adding member: $email with role: $COMMON_ROLE and delivery: $COMMON_DELIVERY_MODE"

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