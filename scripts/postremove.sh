#!/bin/bash
# =========================
# Xray-core > Refract
# Post-removal script
# =========================

set -euo pipefail

SERVICE_NAME="refract.service"

if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
  if systemctl is-enabled --quiet "$SERVICE_NAME"; then
    echo "-- Disabling Refract service"
    systemctl disable "$SERVICE_NAME" || true
  fi

  systemctl daemon-reload || true
else
  echo "info: systemd not detected, skipping disable/daemon-reload"
fi

echo "✓ Refract post-removal script execution completed"
