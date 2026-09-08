#!/bin/bash
# Устанавливает и (пере)запускает smart-house как launchd-агент пользователя.
# Запускать на сервере из каталога ~/smart-house, где уже лежат бинарник
# smart-house, .env и deploy/com.neckita39.smart-house.plist
# (это делает `make deploy DEPLOY=user@host` из основного репозитория).
set -euo pipefail

LABEL=com.neckita39.smart-house
PLIST_SRC="deploy/${LABEL}.plist"
PLIST_DST="$HOME/Library/LaunchAgents/${LABEL}.plist"

chmod +x smart-house

mkdir -p "$HOME/Library/LaunchAgents"
sed "s#__HOME__#$HOME#g" "$PLIST_SRC" > "$PLIST_DST"

# Выгрузить, если уже загружен (первый деплой — не загружен, ошибку игнорируем).
launchctl bootout "gui/$(id -u)/${LABEL}" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$PLIST_DST"

PORT=$(grep '^PORT=' .env | cut -d= -f2)
PORT=${PORT:-8080}

sleep 1
echo "проверяю http://127.0.0.1:${PORT}/api/auth/status ..."
if curl -sf "http://127.0.0.1:${PORT}/api/auth/status"; then
	echo
	echo "OK: сервер отвечает на порту ${PORT}"
else
	echo
	echo "сервер не отвечает на порту ${PORT}; смотрите лог: ~/smart-house/smart-house.log" >&2
	exit 1
fi
