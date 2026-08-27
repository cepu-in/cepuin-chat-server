#!/bin/bash

set -e

SERVER="root@srv1433323"
SERVER_PATH="/var/www/cepuin_chat"
BRANCH="main"

echo "========================================"
echo "       CEPUIN CHAT SERVER DEPLOY"
echo "========================================"

echo ""
echo "[1/4] Checking git status..."
git status

echo ""
echo "[2/4] Pushing to GitHub..."
git push origin "$BRANCH"

echo ""
echo "[3/4] Pulling latest code on server..."

ssh "$SERVER" << EOF
    set -e

    cd "$SERVER_PATH"

    echo "Server directory:"
    pwd

    echo ""
    echo "Pulling GitHub..."
    git pull origin "$BRANCH"

    echo ""
    echo "Current commit:"
    git log -1 --oneline

    echo ""
    echo "Checking .env..."
    if [ ! -f .env ]; then
        echo "ERROR: .env tidak ditemukan!"
        exit 1
    fi

    echo ""
    echo "Deployment source updated."
EOF

echo ""
echo "========================================"
echo "       DEPLOYMENT FINISHED"
echo "========================================"