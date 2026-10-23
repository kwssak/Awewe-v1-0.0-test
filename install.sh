cd /root/awewe-panel  # или где у тебя клон репо

cat > install.sh << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

REPO="https://github.com/kwssak/Awewe-v1-0.0-test.git"
APP="/opt/awewe-panel"
CONF="/etc/awewe-panel"
BIN="$APP/backend/bin/awewe-panel"
XRAY_BIN="$APP/bin/xray"
XRAY_CFG="$CONF/xray-config.json"
PORT="${AWEWE_PORT:-2053}"

if [ "$PORT" -lt 1024 ] || [ "$PORT" -gt 60000 ]; then
  echo "ERROR: port must be between 1024 and 60000"
  exit 1
fi

echo "[1/8] deps"
apt-get update -y
apt-get install -y git curl wget unzip jq uuidgen

echo "[2/8] go"
if ! command -v go >/dev/null 2>&1; then
  wget -q https://go.dev/dl/go1.23.4.linux-amd64.tar.gz -O /tmp/go.tgz
  rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tgz
  export PATH=$PATH:/usr/local/go/bin
fi
go version

echo "[3/8] xray core"
mkdir -p "$APP/bin" "$CONF" "$APP/web"
if [ ! -x "$XRAY_BIN" ]; then
  wget -q https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip -O /tmp/xray.zip
  unzip -o /tmp/xray.zip -d /tmp/xrayd
  cp /tmp/xrayd/xray "$XRAY_BIN"; chmod +x "$XRAY_BIN"
fi
"$XRAY_BIN" version

echo "[4/8] code"
rm -rf "$APP/backend" "$APP/frontend"
git clone "$REPO" /tmp/awewe-src
cp -r /tmp/awewe-src/backend "$APP/backend"
cp -r /tmp/awewe-src/frontend/dist/. "$APP/web/"

echo "[5/8] build"
cd "$APP/backend"
go mod tidy
go build -trimpath -ldflags="-s -w" -o "$BIN" ./cmd/server

echo "[6/8] config"
WEB_PATH="/$(uuidgen | tr -d '-' | cut -c1-16)"
TOKEN="$(uuidgen | tr -d '-')"
cat > "$CONF/config.json" <<JSON
{
  "port": $PORT,
  "web_path": "$WEB_PATH",
  "api_token": "$TOKEN",
  "xray_config": "$XRAY_CFG",
  "xray_bin": "$XRAY_BIN",
  "db": "$CONF/awewe.db",
  "web_dir": "$APP/web",
  "sub_base": "/sub"
}
JSON
echo '{ "log":{"loglevel":"warning"}, "inbounds":[], "outbounds":[{"protocol":"freedom","tag":"direct"}] }' > "$XRAY_CFG"

echo "[7/8] systemd"
cat > /etc/systemd/system/awewe-panel.service <<UNIT
[Unit]
Description=Awewe Panel
After=network.target

[Service]
Type=simple
Environment=GIN_MODE=release
ExecStart=$BIN -config $CONF/config.json
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable --now awewe-panel
sleep 3
systemctl --no-pager status awewe-panel | head -n 6 || true

echo "[8/8] info"
IP=$(curl -s ifconfig.me)
echo "=================================================="
echo " ВЕБ:    http://$IP:$PORT$WEB_PATH"
echo " ТОКЕН:  $TOKEN"
echo " XRAY:   $("$XRAY_BIN" version 2>/dev/null | head -n1)"
echo "=================================================="
echo "Запомни ТОКЕН — он нужен для входа в панель!"
EOF

chmod +x install.sh
