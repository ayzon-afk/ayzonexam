#!/usr/bin/env bash
# ./update.sh — обновить AYZON EXAM на сервере (этап 7). Порядок:
#   1. прежние образы — под тегом :previous
#   2. скачать новые (их собирает GitHub после зелёного CI на main)
#   3. копия базы before-update
#   4. новые сервер и служба копий; сервер при запуске применяет миграции
#   5. сервер не поднялся за 3 минуты — возвращаются прежние образы. Новая миграция сломала прежнюю
#      версию — ./restore.sh <копия before-update> --yes
# Репетиция на компьютере: EXAM_COMPOSE="docker compose -f compose.yml -f local/compose.local.yml --env-file local/.env" UPDATE_PULL=off
set -euo pipefail
cd "$(dirname "$0")"

COMPOSE="${EXAM_COMPOSE:-docker compose}"
PULL="${UPDATE_PULL:-on}"

# Ждать «healthy»; программа падает и Docker перезапускает её снова и снова — не ждать до конца срока
wait_healthy() {
  local name="$1" seconds="$2" status restarts
  for _ in $(seq 1 "$seconds"); do
    status=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$name" 2>/dev/null || echo none)
    [ "$status" = healthy ] && return 0
    restarts=$(docker inspect -f '{{.RestartCount}}' "$name" 2>/dev/null || echo 0)
    [ "$restarts" -ge 3 ] && return 1
    sleep 1
  done
  return 1
}

# Тег :previous — ДО скачивания: образ без тега Docker удаляет, и вернуть его потом нельзя
keep_previous() {
  local container="$1" ref
  ref=$(docker inspect -f '{{.Config.Image}}' "$container")
  docker tag "$(docker inspect -f '{{.Image}}' "$container")" "${ref%:*}:previous"
}

restore_previous() {
  local container="$1" ref
  ref=$(docker inspect -f '{{.Config.Image}}' "$container")
  docker tag "${ref%:*}:previous" "$ref"
}

echo "1/5 Прежняя версия — под тегом :previous…"
keep_previous ayzon-exam-app
keep_previous ayzon-exam-backup

echo "2/5 Новая версия…"
if [ "$PULL" = on ]; then $COMPOSE pull app backup; else echo "   (скачивание пропущено: UPDATE_PULL=off)"; fi

echo "3/5 Копия базы before-update…"
$COMPOSE exec -T backup backup now before-update

echo "4/5 Новые сервер и служба копий…"
$COMPOSE up -d --no-deps app backup
if ! wait_healthy ayzon-exam-app 180; then
  echo "Сервер не поднялся — возвращаю прежнюю версию"
  $COMPOSE logs --tail 40 app || true
  restore_previous ayzon-exam-app
  restore_previous ayzon-exam-backup
  $COMPOSE up -d --no-deps app backup
  wait_healthy ayzon-exam-app 180 || true
  $COMPOSE exec -T backup backup notify "🚨 обновление не удалось, возвращена прежняя версия; копия before-update сделана" || true
  exit 1
fi

echo "5/5 Готово"
$COMPOSE ps
docker image prune -f >/dev/null || true
