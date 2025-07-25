#!/bin/bash

# Alternative push script using HTTPS

echo "Setting up Gitea remote with HTTPS..."

# Remove existing gitea remote if exists
if git remote | grep -q "gitea"; then
    echo "Removing existing gitea remote..."
    git remote remove gitea
fi

# Add HTTPS remote
echo "Adding Gitea HTTPS remote..."
git remote add gitea https://ci.gmac.io/mackieg/power-grid-backend.git

echo -e "\nRemotes:"
git remote -v

echo -e "\nPushing to Gitea..."
echo "You'll be prompted for your Gitea username and password/token"
git push gitea master:master

if [ $? -eq 0 ]; then
    echo -e "\n✅ Successfully pushed to Gitea!"
    echo -e "\n📋 Next steps:"
    echo "1. Go to: https://ci.gmac.io/mackieg/power-grid-backend"
    echo "2. Navigate to Settings → Actions → Secrets"
    echo "3. Add these secrets:"
    echo "   - GITEA_TOKEN: Your personal access token"
    echo "   - KUBECONFIG: Base64 encoded kubeconfig file"
else
    echo -e "\n❌ Push failed. Please ensure:"
    echo "1. Repository exists at https://ci.gmac.io/mackieg/power-grid-backend"
    echo "2. You have the correct username/password"
    echo "3. Consider using a personal access token instead of password"
fi