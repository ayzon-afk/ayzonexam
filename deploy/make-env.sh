#!/usr/bin/env bash
# ./make-env.sh <поддомен> — создаёт deploy/.env со случайным паролем базы и секретом webhook.
# Уже есть — не трогает. Секреты на экран не выводятся
set -euo pipefail
cd "$(dirname "$0")"

if [ -e .env ]; then
  echo "deploy/.env уже есть — не трогаю. Новый: переименуйте старый и запустите снова"
  exit 0
fi
if [ $# -lt 1 ]; then
  echo "Использование: ./make-env.sh <поддомен>   например: ./make-env.sh exam.ayzon.uz"
  exit 1
fi

# Только hex: пароль входит в адрес postgres://… без экранирования
hex() { openssl rand -hex "$1"; }

umask 077
sed \
  -e "s|^EXAM_DOMAIN=.*|EXAM_DOMAIN=$1|" \
  -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$(hex 24)|" \
  -e "s|^WEBHOOK_SECRET=.*|WEBHOOK_SECRET=$(hex 32)|" \
  .env.example > .env

echo "Создан deploy/.env для $1 (права 600). Сохраните его копию у себя."
echo "Осталось вписать: BOT_TOKEN, ADMIN_IDS, GEMINI_API_KEY, PAY_CARD, PAY_CARD_NAME и R2_* (копии вне сервера)"
