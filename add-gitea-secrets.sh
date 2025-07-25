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

# Add GITEATOKEN (without underscore)
echo -n "Adding GITEATOKEN secret... "
curl -X PUT \
  "${GITEA_URL}/api/v1/repos/${REPO_OWNER}/${REPO_NAME}/actions/secrets/GITEATOKEN" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"data\": \"${GITEA_TOKEN}\"}" \
  -s -o /dev/null -w "%{http_code}"
echo

# Add KUBECONFIG
echo -n "Adding KUBECONFIG secret... "
curl -X PUT \
  "${GITEA_URL}/api/v1/repos/${REPO_OWNER}/${REPO_NAME}/actions/secrets/KUBECONFIG" \
  -H "Authorization: token ${GITEA_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"data\": \"${KUBECONFIG_BASE64}\"}" \
  -s -o /dev/null -w "%{http_code}"
echo

echo -e "\n✅ Secrets added! Check the Actions tab at:"
echo "${GITEA_URL}/${REPO_OWNER}/${REPO_NAME}/actions"