#!/bin/bash

# Push script to handle Gitea deployment from go_server directory

echo "Preparing Power Grid Backend for Gitea deployment..."

# First, let's see what needs to be committed
echo -e "\nChecking for deployment files..."

# List files that should exist for deployment
REQUIRED_FILES=(
    "Dockerfile"
    "../k8s/deployment.yaml"
    "../k8s/ingress.yaml"
    "../.github/workflows/deploy.yml"
)

echo -e "\nVerifying required files:"
for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ $file"
    else
        echo "✗ $file (missing)"
    fi
done

# Show current git status
echo -e "\nCurrent git status:"
git status --short

# Stage the Dockerfile if needed
if git status --porcelain | grep -q "Dockerfile"; then
    echo -e "\nStaging Dockerfile..."
    git add Dockerfile
fi

# Commit if there are staged changes
if ! git diff --cached --quiet; then
    echo -e "\nCommitting deployment configuration..."
    git commit -m "Add Kubernetes deployment configuration for Gitea CI/CD

- Add multi-stage Dockerfile for Go server
- Configure deployment for ws.power-grid.gmac.io
- Set up CI/CD workflow for automated deployments"
fi

# Add Gitea remote if not exists
if ! git remote | grep -q "gitea"; then
    echo -e "\nAdding Gitea remote..."
    git remote add gitea git@ci.gmac.io:mackieg/power-grid-backend.git
else
    echo -e "\nGitea remote already exists"
fi

# Show remotes
echo -e "\nRemotes:"
git remote -v

# Push to Gitea
echo -e "\nPushing to Gitea (master branch)..."
git push gitea master:master

if [ $? -eq 0 ]; then
    echo -e "\n✅ Successfully pushed to Gitea!"
    echo -e "\n📋 Next steps:"
    echo "1. Go to: https://ci.gmac.io/mackieg/power-grid-backend"
    echo "2. Navigate to Settings → Actions → Secrets"
    echo "3. Add these secrets:"
    echo "   - GITEA_TOKEN: Your personal access token"
    echo "   - KUBECONFIG: Base64 encoded kubeconfig file"
    echo -e "\n🚀 Once secrets are added, the deployment will start automatically!"
else
    echo -e "\n❌ Push failed. Please check:"
    echo "1. SSH access to ci.gmac.io"
    echo "2. Repository exists at ci.gmac.io/mackieg/power-grid-backend"
    echo "3. You have push permissions"
fi