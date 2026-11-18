#!/usr/bin/env bash
# ./restore.sh                 — список копий (на сервере и в R2)
# ./restore.sh <файл> --yes    — база целиком из копии: сервер останавливается, база пересоздаётся, сервер
#                                запускается. Копии нет на сервере — сначала скачивается из R2.
# Всё, что записано после копии, пропадает — перед восстановлением делается копия manual текущей базы.
# Репетиция на компьютере: EXAM_COMPOSE="docker compose -f compose.yml -f local/compose.local.yml --env-file local/.env"
set -euo pipefail
cd "$(dirname "$0")"

COMPOSE="${EXAM_COMPOSE:-docker compose}"
bk() { $COMPOSE exec -T backup backup "$@"; }

if [ $# -eq 0 ]; then
  bk list
  exit 0
fi
file=$(basename "$1")
if [ "${2:-}" != --yes ]; then
  echo "Восстановить базу из $file? Всё после копии пропадёт. Запустите: ./restore.sh $file --yes"
  exit 1
fi

if ! $COMPOSE exec -T backup test -f "/backups/$file"; then
  bk fetch "$file"
fi
echo "Копия текущей базы перед восстановлением…"
bk now manual
$COMPOSE stop app
status=0
bk restore "$file" || status=$?
$COMPOSE start app
if [ "$status" -ne 0 ]; then
  echo "Восстановление не удалось (код $status). Текущая база — в копии manual выше"
  exit "$status"
fi
echo "Готово: база из $file, сервер запущен"
