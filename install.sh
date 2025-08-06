#!/bin/bash -e

resolve_current_dir() {
  SOURCE="${BASH_SOURCE[0]}"
  while [ -h "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
  done

  cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd
}

join_paths() {
  local base="${1%/}"
  local sub="${2#/}"
  printf '%s/%s\n' "$base" "$sub"
}

UNIT_FILE="/etc/systemd/system/recorder.service"
BIN_NAME="recorder"

if [[ ! -f "./$BIN_NAME" ]]; then
  echo "Error: '$BIN_NAME' not found in the current directory."
  exit 1
fi

DEFAULT_SERVICE_USER="${USER:-$(whoami)}"
DEFAULT_INSTALL_DIR_PATH="/usr/local/bin/"
DEFAULT_CONFIG_PATH="$(resolve_current_dir)/config.yaml"
DEFAULT_VIDEO_DIR_PATH="$(resolve_current_dir)/videos"

echo "Enter the user to run the service as:"
read -e -p "You may use a colon to specify user:group. [${DEFAULT_SERVICE_USER}]: " SERVICE_USER
SERVICE_USER="${SERVICE_USER:-$DEFAULT_SERVICE_USER}"
read -e -p "Enter installation path [${DEFAULT_INSTALL_DIR_PATH}]: " INSTALL_DIR_PATH
INSTALL_DIR_PATH="${INSTALL_DIR_PATH:-$DEFAULT_INSTALL_DIR_PATH}"
read -e -p "Enter configuration file path [${DEFAULT_CONFIG_PATH}]: " CONFIG_PATH
CONFIG_PATH="${CONFIG_PATH:-$DEFAULT_CONFIG_PATH}"
read -e -p "Enter video directory path [${DEFAULT_VIDEO_DIR_PATH}]: " VIDEO_DIR_PATH
VIDEO_DIR_PATH="${VIDEO_DIR_PATH:-$DEFAULT_VIDEO_DIR_PATH}"

INSTALL_BIN_PATH="$(join_paths "$INSTALL_DIR_PATH" "$BIN_NAME")"

echo -e "\nInstallation details:"
echo "Service user: $SERVICE_USER"
echo "Installation binary path: $INSTALL_BIN_PATH"
echo "Configuration file path: $CONFIG_PATH"
echo "Video directory path: $VIDEO_DIR_PATH"
echo -e "\nThis script will install the recorder binary and service."

read -e -p "Proceed with installation? (y/N): " PROCEED
if [[ "$PROCEED" != "y" && "$PROCEED" != "Y" ]]; then
  echo "Installation aborted."
  exit 1
fi

cp ./recorder "$INSTALL_BIN_PATH"

if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "Warning: Configuration file '$CONFIG_PATH' does not exist. It will be created during the installation."
  
  cat > "$CONFIG_PATH" <<EOF
cameras:
EOF
  
  chown "$SERVICE_USER" "$CONFIG_PATH"
fi

if [[ ! -d "$VIDEO_DIR_PATH" ]]; then
  mkdir -p "$VIDEO_DIR_PATH"
  chown -R "$SERVICE_USER" "$VIDEO_DIR_PATH"
fi

if [[ "$SERVICE_USER" == *:* ]]; then
  SERVICE_GROUP="${SERVICE_USER#*:}"
  SERVICE_USER="${SERVICE_USER%%:*}"
else
  SERVICE_GROUP="$SERVICE_USER"
fi

ALREADY_INSTALLED=$(systemctl list-unit-files | grep -q "recorder.service" && echo "yes" || echo "no")

cat > "$UNIT_FILE" <<EOF
[Unit]
Description=recorder service
After=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_GROUP
ExecStart=/usr/bin/env "$INSTALL_BIN_PATH" -config "$CONFIG_PATH"
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
WorkingDirectory=$VIDEO_DIR_PATH

[Install]
WantedBy=multi-user.target
EOF

if [[ "$ALREADY_INSTALLED" == "yes" ]]; then
  echo "Updating existing service..."
  systemctl daemon-reload
  systemctl restart recorder.service
else
  echo "Installing new service..."
  systemctl enable recorder.service
  systemctl start recorder.service
fi

echo "Installation complete."

