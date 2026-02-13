#!/bin/bash
# =========================
# Xray-core > Refract
# Pre-removal script
# =========================

set -euo pipefail

SERVICE_NAME="refract.service"

if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "-- Stopping Refract service"
    systemctl stop "$SERVICE_NAME" || true
  fi
else
  echo "info: systemd not detected, skipping service stop"
fi

echo "✓ Refract pre-removal script execution completed"
