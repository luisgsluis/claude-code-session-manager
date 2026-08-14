#!/usr/bin/env bash
# Reconstruye e instala ccsm-agent (el binario que corre en el host) de forma
# que el servicio nunca quede parado a medias si algo interrumpe el proceso
# (sesión de Claude reiniciada, terminal cerrada, etc.).
#
# Claves de por qué está hecho así, no como un stop; cp; start manual:
# - El binario se sustituye con `install` (rename atómico) SIN parar el
#   servicio antes: en Linux el proceso ya en marcha sigue usando el inodo
#   viejo aunque el fichero cambie de contenido, así que la sustitución no
#   afecta al agente que sigue corriendo.
# - El único punto en que el servicio deja de estar activo es el propio
#   `systemctl restart`, una sola llamada atómica de systemd — no hay una
#   ventana de "parado" que pueda quedar a medias entre dos tool calls.
# - El `trap` en EXIT es la red de seguridad final: si el script muere en
#   cualquier punto (build roto, Ctrl-C, sesión cortada a media ejecución),
#   al salir comprueba igualmente que el servicio esté activo y lo arranca
#   si no lo está.
set -euo pipefail

cd "$(dirname "$0")/.."

trap '
  if ! systemctl is-active --quiet ccsm-agent; then
    echo "[deploy-agent] recuperando: ccsm-agent no está activo, arrancando..." >&2
    sudo systemctl start ccsm-agent || true
  fi
' EXIT

echo "[deploy-agent] build..."
make agent

echo "[deploy-agent] instalando binario (rename atómico, sin tocar el proceso en marcha)..."
sudo install -m 0755 bin/ccsm-agent /usr/local/bin/ccsm-agent

echo "[deploy-agent] restart..."
sudo systemctl restart ccsm-agent

sleep 1
if ! systemctl is-active --quiet ccsm-agent; then
  echo "[deploy-agent] ERROR: ccsm-agent no quedó activo tras el restart" >&2
  journalctl -u ccsm-agent -n 30 --no-pager >&2
  exit 1
fi

echo "[deploy-agent] OK: ccsm-agent activo ($(systemctl show -p MainPID --value ccsm-agent))"
