#!/bin/bash

# Script to add secrets to Gitea via API

GITEA_URL="https://ci.gmac.io"
REPO_OWNER="mackieg"
REPO_NAME="power-grid-backend"

# Prompt for token
echo -n "Enter your Gitea personal access token: "
read -s GITEA_TOKEN
echo

# Read kubeconfig from file
KUBECONFIG_BASE64=$(cat kubeconfig-base64.txt)

echo "Adding secrets to Gitea repository..."

# First, let's check if the API endpoint is accessible
echo "Testing API access..."
API_TEST=$(curl -s -o /dev/null -w "%{http_code}" \
  "${GITEA_URL}/api/v1/repos/${REPO_OWNER}/${REPO_NAME}" \
  -H "Authorization: Bearer ${GITEA_TOKEN}")

if [ "$API_TEST" != "200" ]; then
  echo "❌ API access failed (HTTP $API_TEST). Please check:"
  echo "   - Your personal access token is correct"
  echo "   - The token has the required permissions"
  echo "   - Try using 'Bearer' instead of 'token' in Authorization header"
  exit 1
fi

# Note: The exact API endpoint for Gitea Actions secrets might be different
# Let's try the GitHub-compatible endpoint format
echo -e "\nAdding secrets..."

# Add GITEATOKEN (without underscore)
echo -n "Adding GITEATOKEN secret... "
RESPONSE=$(curl -X PUT \
  "${GITEA_URL}/api/v1/repos/${REPO_OWNER}/${REPO_NAME}/actions/secrets/GITEATOKEN" \
  -H "Authorization: Bearer ${GITEA_TOKEN}" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d "{\"encrypted_value\": \"${GITEA_TOKEN}\", \"key_id\": \"\"}" \
  -s -w "\n%{http_code}")
echo "$RESPONSE" | tail -1

# Add KUBECONFIG
echo -n "Adding KUBECONFIG secret... "
RESPONSE=$(curl -X PUT \
  "${GITEA_URL}/api/v1/repos/${REPO_OWNER}/${REPO_NAME}/actions/secrets/KUBECONFIG" \
  -H "Authorization: Bearer ${GITEA_TOKEN}" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d "{\"encrypted_value\": \"${KUBECONFIG_BASE64}\", \"key_id\": \"\"}" \
  -s -w "\n%{http_code}")
echo "$RESPONSE" | tail -1

echo -e "\nNote: If you see 404 errors, you may need to add secrets manually via the web UI:"
echo "${GITEA_URL}/${REPO_OWNER}/${REPO_NAME}/settings/actions/secrets"
echo -e "\nSecrets to add:"
echo "1. GITEATOKEN = Your personal access token"
echo "2. KUBECONFIG = Contents of kubeconfig-base64.txt"