#!/bin/bash
# =========================
# Xray-core > Refract
# Post-installation script
# =========================

set -euo pipefail

SERVICE_NAME="refract.service"
BINARY_PATH="/opt/refract/refract"
DEFAULTS_FILE="/etc/default/refract"

# Ensure binary exists
if [[ -f "$BINARY_PATH" ]]; then
  chmod 755 "$BINARY_PATH"
else
  echo "warn: Binary not found at $BINARY_PATH, skipping chmod"
fi

# Ensure defaults file exists
if [[ -f "$DEFAULTS_FILE" ]]; then
  chmod 644 "$DEFAULTS_FILE"
else
  echo "warn: Defaults file not found at $DEFAULTS_FILE, skipping chmod"
fi

# Only try systemctl if systemd is present
if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
  systemctl daemon-reload || true
  if [[ -f "/etc/systemd/system/$SERVICE_NAME" ]]; then
    # Some distros/versions may exit non-zero even if the unit is enabled, so ignore failures here.
    systemctl enable "$SERVICE_NAME" || true
  else
    echo "warn: Systemd unit /etc/systemd/system/$SERVICE_NAME not found, skipping enable"
  fi
else
  echo "info: systemd not detected, skipping systemctl daemon-reload/enable"
fi

cat <<'EOF'
✓ Refract service successfully installed and (if systemd is available) enabled for automatic startup during system boot sequence

╔════════════════════════════════════════════════════════════════════════════╗
║                            !!! IMPORTANT NOTICE !!!                        ║
║                                                                            ║
║  Before starting the service, you MUST:                                    ║
║                                                                            ║
║  1. Configure the correct log file path in /etc/default/refract            ║
║     Current default path may not work for your setup!                      ║
║                                                                            ║
║  2. Review the documentation at:                                           ║
║     https://github.com/quyxishi/refract                                    ║
║                                                                            ║
║  3. Configure additional parameters according to your needs                ║
║                                                                            ║
║  After configuration is complete, start the service with:                  ║
║  systemctl start refract.service                                           ║
║                                                                            ║
╚════════════════════════════════════════════════════════════════════════════╝
EOF
