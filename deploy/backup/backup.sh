#!/bin/sh
# Служба копий и наблюдения AYZON EXAM (этап 7). Команды:
#   run                 — работать всё время: каждую минуту проверять сервер, в BACKUP_TIME — ночная копия
#   now [вид]           — копия сейчас (вид: manual, before-update, nightly)
#   list                — копии на сервере и в R2
#   fetch <файл>        — скачать копию из R2 в /backups
#   restore <файл>      — база целиком из копии (сервер должен быть остановлен — это делает deploy/restore.sh)
#   notify <текст>      — сообщение владельцу в Telegram
# Копии: pg_dump -Fc в /backups, после записи читаются pg_restore --list — битая копия не считается сделанной.
# На сервере хранятся 7 ночных и 5 перед обновлением; в R2 — ночные 14 дней, воскресные 8 недель.
set -eu

# «UTC-5» в POSIX — это UTC+5 (Ташкент); так busybox date считает время без tzdata
export TZ=UTC-5
DIR=/backups
DB=ayzonexam
BACKUP_TIME=${BACKUP_TIME:-03:00}
APP_HEALTH=${APP_HEALTH:-http://ayzon-exam-app:8080/api/health}

r2_on() { [ -n "${R2_BUCKET:-}" ] && [ -n "${R2_ENDPOINT:-}" ]; }

# rclone берёт настройки удалённого хранилища «r2» из переменных RCLONE_CONFIG_R2_*
r2_env() {
  export RCLONE_CONFIG_R2_TYPE=s3 RCLONE_CONFIG_R2_PROVIDER=Cloudflare \
    RCLONE_CONFIG_R2_ENDPOINT="$R2_ENDPOINT" RCLONE_CONFIG_R2_ACCESS_KEY_ID="${R2_ACCESS_KEY_ID:-}" \
    RCLONE_CONFIG_R2_SECRET_ACCESS_KEY="${R2_SECRET_ACCESS_KEY:-}" RCLONE_CONFIG_R2_NO_CHECK_BUCKET=true
}

# Сообщение всем из ADMIN_IDS. Ошибку curl не печатать: в адресе Bot API — токен
notify() {
  [ -n "${BOT_TOKEN:-}" ] || { echo "уведомление (BOT_TOKEN не задан): $1"; return 0; }
  for id in $(echo "${ADMIN_IDS:-}" | tr ',' ' '); do
    curl -s -o /dev/null --max-time 15 "https://api.telegram.org/bot$BOT_TOKEN/sendMessage" \
      --data-urlencode "chat_id=$id" --data-urlencode "text=AYZON EXAM: $1" || echo "уведомление не отправлено"
  done
}

backup() {
  kind=${1:-manual}
  case "$kind" in manual|before-update|nightly) ;; *) echo "вид копии: manual, before-update или nightly"; return 1 ;; esac
  mkdir -p "$DIR"
  name="$DB-$(date +%Y-%m-%d_%H%M)-$kind.dump"
  tmp="$DIR/.$name.part"
  if ! pg_dump -Fc -Z 6 -d "$DB" -f "$tmp" || ! pg_restore --list "$tmp" >/dev/null; then
    rm -f "$tmp"
    notify "🚨 копия базы ($kind) не сделана"
    return 1
  fi
  mv "$tmp" "$DIR/$name"
  echo "копия: $name ($(du -h "$DIR/$name" | cut -f1))"
  # На сервере — последние 7 ночных и 5 перед обновлением; ручные удаляются руками
  for k in nightly:7 before-update:5; do
    ls -1t "$DIR"/*-"${k%%:*}".dump 2>/dev/null | tail -n +"$((${k##*:} + 1))" | xargs -r rm -f
  done
  if r2_on; then
    r2_env
    dest=daily
    [ "$kind" = nightly ] && [ "$(date +%u)" = 7 ] && dest=weekly
    if rclone copyto "$DIR/$name" "r2:$R2_BUCKET/$dest/$name" --s3-no-check-bucket -q; then
      rclone delete "r2:$R2_BUCKET/daily" --min-age 14d -q || true
      rclone delete "r2:$R2_BUCKET/weekly" --min-age 56d -q || true
    else
      notify "⚠️ копия $name сделана на сервере, но в R2 не ушла"
    fi
  fi
}

list() {
  echo "На сервере:"
  ls -1t "$DIR"/*.dump 2>/dev/null | while read -r f; do echo "  $(basename "$f")  $(du -h "$f" | cut -f1)"; done
  if r2_on; then
    r2_env
    echo "В R2:"
    rclone lsf -R "r2:$R2_BUCKET" --files-only | sed 's/^/  /'
  fi
}

fetch() {
  r2_on || { echo "R2 не настроен"; return 1; }
  r2_env
  base=$(basename "$1")
  for d in daily weekly; do
    if rclone copyto "r2:$R2_BUCKET/$d/$base" "$DIR/$base" -q 2>/dev/null; then echo "скачано: $base"; return 0; fi
  done
  echo "в R2 нет $base"
  return 1
}

restore() {
  f="$DIR/$(basename "$1")"
  [ -f "$f" ] || { echo "нет копии $f (список: list, из R2: fetch)"; return 1; }
  pg_restore --list "$f" >/dev/null
  # Базу пересоздать целиком: копия — вся схема и данные, включая таблицу миграций
  psql -v ON_ERROR_STOP=1 -q -d postgres -c "DROP DATABASE IF EXISTS $DB WITH (FORCE)" -c "CREATE DATABASE $DB"
  pg_restore --no-owner --exit-on-error -d "$DB" "$f"
  echo "база восстановлена из $(basename "$f"): пользователей $(psql -tAq -d "$DB" -c 'select count(*) from users')"
}

watch() {
  fails=0
  down=0
  last=""
  echo "наблюдение: $APP_HEALTH; ночная копия в $BACKUP_TIME (Ташкент)"
  while true; do
    if wget -qO /dev/null -T 5 "$APP_HEALTH" 2>/dev/null; then
      [ "$down" = 1 ] && notify "✅ сервер снова отвечает"
      fails=0
      down=0
    else
      fails=$((fails + 1))
      # Три минуты подряд без ответа — сообщить один раз, а не каждую минуту
      if [ "$fails" = 3 ]; then
        down=1
        notify "🚨 сервер не отвечает 3 минуты"
      fi
    fi
    now=$(date +%H:%M)
    today=$(date +%F)
    if [ "$now" = "$BACKUP_TIME" ] && [ "$last" != "$today" ]; then
      last=$today
      backup nightly || true
    fi
    sleep 60
  done
}

cmd=${1:-run}
[ $# -gt 0 ] && shift
case "$cmd" in
  run) watch ;;
  now) backup "${1:-manual}" ;;
  list) list ;;
  fetch) [ $# = 1 ] || { echo "fetch <файл>"; exit 1; }; fetch "$1" ;;
  restore) [ $# = 1 ] || { echo "restore <файл>"; exit 1; }; restore "$1" ;;
  notify) notify "$*" ;;
  *) sed -n '2,9p' "$0"; exit 1 ;;
esac
