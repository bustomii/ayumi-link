#!/usr/bin/env bash
# Kelola service systemd (Linux) ayumi-link. Padanan LaunchAgent di macOS.
# Root: unit system di /etc/systemd/system. User biasa: unit user (~/.config/systemd/user).
set -eu
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
NAME="ayumi-link"
RUN="${ROOT}/scripts/native-service-run.sh"

command -v systemctl >/dev/null || { echo "systemctl tidak ada: distro ini tidak memakai systemd (pakai: make build)" >&2; exit 1; }

if [ "$(id -u)" -eq 0 ]; then
  UNIT="/etc/systemd/system/${NAME}.service"
  CTL=(systemctl)
  LOGS=(journalctl -u "$NAME")
  TARGET="multi-user.target"
else
  UNIT="${HOME}/.config/systemd/user/${NAME}.service"
  CTL=(systemctl --user)
  LOGS=(journalctl --user -u "$NAME")
  TARGET="default.target"
fi

write_unit() {
  mkdir -p "$(dirname "$UNIT")"
  chmod +x "$RUN" "${ROOT}/scripts/native-build.sh"
  cat >"$UNIT" <<UNITEOF
[Unit]
Description=Ayumi Link (make build lalu serve)
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
WorkingDirectory=${ROOT}
ExecStart=/bin/bash "${RUN}"
Environment="HOME=${HOME}"
Environment=LANG=en_US.UTF-8
Restart=always
RestartSec=30

[Install]
WantedBy=${TARGET}
UNITEOF
  "${CTL[@]}" daemon-reload
}

need_unit() {
  [ -f "$UNIT" ] || { echo "Belum terpasang. Jalankan: make service-install" >&2; exit 1; }
}

case "${1:-}" in
  install)
    write_unit
    "${CTL[@]}" enable "$NAME"
    "${CTL[@]}" restart "$NAME"
    echo "Installed + started: ${NAME} (${UNIT})"
    echo "UI: http://127.0.0.1:8081 (setelah build selesai)"
    echo "Log: make service-logs"
    if [ "$(id -u)" -ne 0 ]; then
      echo "Agar tetap jalan setelah logout/reboot: sudo loginctl enable-linger $(id -un)"
    fi
    ;;
  uninstall)
    "${CTL[@]}" disable --now "$NAME" 2>/dev/null || true
    rm -f "$UNIT"
    "${CTL[@]}" daemon-reload
    echo "Uninstalled ${NAME}"
    ;;
  start) need_unit; "${CTL[@]}" start "$NAME" ;;
  stop) need_unit; "${CTL[@]}" stop "$NAME"; echo "Stopped ${NAME}" ;;
  restart)
    need_unit
    write_unit
    "${CTL[@]}" restart "$NAME"
    echo "Restarted ${NAME} → make build jalan lagi"
    ;;
  status) "${CTL[@]}" status "$NAME" --no-pager ;;
  logs) "${LOGS[@]}" -n 50 -f ;;
  *) echo "Usage: sh scripts/native-service.sh <install|uninstall|start|stop|restart|status|logs>"; exit 1 ;;
esac
