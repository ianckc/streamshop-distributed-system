#!/bin/sh
set -eu

npx srvx serve \
  --prod \
  --host 127.0.0.1 \
  --port 3020 \
  --static ./dist/client \
  --entry ./dist/server/server.js &

# Wait until SSR accepts connections before nginx starts answering Traefik.
i=0
while [ "$i" -lt 60 ]; do
  if wget -qO- http://127.0.0.1:3020/admin/ >/dev/null 2>&1; then
    break
  fi
  i=$((i + 1))
  sleep 0.25
done

exec nginx -g 'daemon off;'
