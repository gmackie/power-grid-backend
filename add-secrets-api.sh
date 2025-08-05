#!/bin/bash

# Add secrets to Gitea using the correct API

GITEA_URL="https://ci.gmac.io"
REPO_OWNER="mackieg"
REPO_NAME="power-grid-backend"

# Prompt for token
echo -n "Enter your Gitea personal access token: "
read -s GITEA_TOKEN
echo

# Read kubeconfig from file
KUBECONFIG_BASE64=$(cat kubeconfig-base64.txt)

echo "Testing API access..."
# Test API access
curl -s "${GITEA_URL}/api/v1/user" \
  -H "Authorization: token ${GITEA_TOKEN}" | jq -r .login || echo "API test failed"

# Create or update secrets
echo -e "\nAdding GITEATOKEN secret..."
curl -X PUT \
  "${GITEA_URL}/api/v1/repos/${REPO_OWNER}/${REPO_NAME}/actions/secrets/GITEATOKEN" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"data\": \"${GITEA_TOKEN}\"}"

echo -e "\n\nAdding KUBECONFIG secret..."
curl -X PUT \
  "${GITEA_URL}/api/v1/repos/${REPO_OWNER}/${REPO_NAME}/actions/secrets/KUBECONFIG" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"data\": \"${KUBECONFIG_BASE64}\"}"

echo -e "\n\n✅ Done! Check: ${GITEA_URL}/${REPO_OWNER}/${REPO_NAME}/actions"