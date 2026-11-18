# Служба копий и наблюдения (этап 7): pg_dump той же версии, что база, rclone для R2, curl для Telegram
FROM postgres:17-alpine
RUN apk add --no-cache rclone curl
COPY backup.sh /usr/local/bin/backup
RUN chmod +x /usr/local/bin/backup
ENTRYPOINT ["backup"]
CMD ["run"]
