#!/bin/bash

set -e

SERVER_PATH="/var/www/cepuin_chat"
BRANCH="main"

echo "========================================"
echo "       CEPUIN CHAT SERVER DEPLOY"
echo "========================================"

echo ""
echo "[1/5] Checking server directory..."
cd "$SERVER_PATH"

echo "Current directory:"
pwd

echo ""
echo "[2/5] Pulling latest code from GitHub..."
git pull origin "$BRANCH"

echo ""
echo "[3/5] Checking environment..."

if [ ! -f .env ]; then
    echo "ERROR: .env tidak ditemukan!"
    exit 1
fi

echo ".env found."

echo ""
echo "[4/5] Updating Go dependencies..."
go mod download

echo ""
echo "[5/5] Restarting Air..."

pkill -f "air" || true

sleep 2

nohup air > /var/log/cepuin_chat_air.log 2>&1 &

sleep 3

echo ""
echo "========================================"
echo "       DEPLOYMENT FINISHED"
echo "========================================"

echo ""
echo "Checking port 8081..."

ss -ltnp | grep ':8081' || true

echo ""
echo "Health check..."

curl -i http://127.0.0.1:8081/health