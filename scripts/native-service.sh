#!/usr/bin/env bash
# Kelola launchd user agent: com.ayumi.link
# Restart = kill proses + jalankan lagi scripts/native-service-run.sh → make build.
set -eu
# Linux memakai systemd; sisanya di bawah khusus launchd (macOS).
if [ "$(uname -s)" = "Linux" ]; then
  exec bash "$(dirname "$0")/native-service-systemd.sh" "$@"
fi
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
LABEL="com.ayumi.link"
UID_NUM=$(id -u)
DOMAIN="gui/${UID_NUM}"
PLIST="${HOME}/Library/LaunchAgents/${LABEL}.plist"
LOG_DIR="${HOME}/Library/Logs/ayumi-link"
RUN="${ROOT}/scripts/native-service-run.sh"

usage() {
  cat <<EOF
Usage: sh scripts/native-service.sh <install|uninstall|start|stop|restart|status|logs>

  install    Pasang LaunchAgent + start (RunAtLoad)
  uninstall  Stop + hapus plist
  start      bootstrap / kickstart
  stop       bootout
  restart    kickstart -k  → make build jalan lagi
  status     launchctl print ringkas
  logs       tail -f stdout/stderr service
EOF
}

write_plist() {
  mkdir -p "$(dirname "$PLIST")" "$LOG_DIR"
  chmod +x "$RUN" "${ROOT}/scripts/native-build.sh" "${ROOT}/scripts/native-service.sh"
  cat >"$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${LABEL}</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/bash</string>
    <string>${RUN}</string>
  </array>
  <key>WorkingDirectory</key>
  <string>${ROOT}</string>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>ThrottleInterval</key>
  <integer>30</integer>
  <key>StandardOutPath</key>
  <string>${LOG_DIR}/stdout.log</string>
  <key>StandardErrorPath</key>
  <string>${LOG_DIR}/stderr.log</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>HOME</key>
    <string>${HOME}</string>
    <key>LANG</key>
    <string>en_US.UTF-8</string>
  </dict>
</dict>
</plist>
EOF
}

loaded() {
  launchctl print "${DOMAIN}/${LABEL}" >/dev/null 2>&1
}

cmd_install() {
  write_plist
  if loaded; then
    launchctl bootout "${DOMAIN}/${LABEL}" 2>/dev/null || true
    sleep 1
  fi
  launchctl bootstrap "${DOMAIN}" "$PLIST"
  launchctl enable "${DOMAIN}/${LABEL}" 2>/dev/null || true
  launchctl kickstart -k "${DOMAIN}/${LABEL}"
  echo "Installed + started: ${LABEL}"
  echo "UI: http://127.0.0.1:8081"
  echo "Logs: ${LOG_DIR}/"
  echo "Redeploy: make service-restart   (atau: sh scripts/native-service.sh restart)"
}

cmd_uninstall() {
  if loaded; then
    launchctl bootout "${DOMAIN}/${LABEL}" 2>/dev/null || true
  fi
  rm -f "$PLIST"
  echo "Uninstalled ${LABEL}"
}

cmd_start() {
  if [ ! -f "$PLIST" ]; then
    echo "Belum terpasang. Jalankan: make service-install" >&2
    exit 1
  fi
  if loaded; then
    launchctl kickstart -k "${DOMAIN}/${LABEL}"
  else
    launchctl bootstrap "${DOMAIN}" "$PLIST"
    launchctl kickstart -k "${DOMAIN}/${LABEL}"
  fi
  echo "Started ${LABEL} (make build…)"
}

cmd_stop() {
  if loaded; then
    launchctl bootout "${DOMAIN}/${LABEL}" 2>/dev/null || true
    echo "Stopped ${LABEL}"
  else
    echo "Not loaded"
  fi
}

cmd_restart() {
  if [ ! -f "$PLIST" ]; then
    echo "Belum terpasang. Jalankan: make service-install" >&2
    exit 1
  fi
  # Tulis ulang plist jika path repo / script berubah.
  write_plist
  if loaded; then
    launchctl bootout "${DOMAIN}/${LABEL}" 2>/dev/null || true
    sleep 1
  fi
  # Kadang bootout menyisakan state; bootstrap gagal → coba enable + kickstart / install ulang.
  if ! launchctl bootstrap "${DOMAIN}" "$PLIST" 2>/dev/null; then
    launchctl bootout "${DOMAIN}/${LABEL}" 2>/dev/null || true
    sleep 1
    launchctl bootstrap "${DOMAIN}" "$PLIST" || {
      echo "bootstrap gagal; coba: make service-uninstall && make service-install" >&2
      exit 1
    }
  fi
  launchctl enable "${DOMAIN}/${LABEL}" 2>/dev/null || true
  launchctl kickstart -k "${DOMAIN}/${LABEL}"
  echo "Restarted ${LABEL} → make build jalan lagi"
  echo "Logs: tail -f ${LOG_DIR}/stdout.log"
}

cmd_status() {
  if loaded; then
    launchctl print "${DOMAIN}/${LABEL}" 2>/dev/null | head -40
  else
    echo "Not loaded (${PLIST} $([ -f "$PLIST" ] && echo exists || echo missing))"
    exit 1
  fi
}

cmd_logs() {
  mkdir -p "$LOG_DIR"
  touch "${LOG_DIR}/stdout.log" "${LOG_DIR}/stderr.log"
  echo "=== ${LOG_DIR} ==="
  tail -n 50 -f "${LOG_DIR}/stdout.log" "${LOG_DIR}/stderr.log"
}

case "${1:-}" in
  install) cmd_install ;;
  uninstall) cmd_uninstall ;;
  start) cmd_start ;;
  stop) cmd_stop ;;
  restart) cmd_restart ;;
  status) cmd_status ;;
  logs) cmd_logs ;;
  *) usage; exit 1 ;;
esac
