#!/bin/sh
set -eu

/usr/local/bin/shiro-api &
api_pid=$!

nginx -g 'daemon off;' &
nginx_pid=$!

shutdown() {
  kill -TERM "$api_pid" "$nginx_pid" 2>/dev/null || true
  wait "$api_pid" 2>/dev/null || true
  wait "$nginx_pid" 2>/dev/null || true
}

trap shutdown INT TERM

while kill -0 "$api_pid" 2>/dev/null && kill -0 "$nginx_pid" 2>/dev/null; do
  sleep 1
done

if ! kill -0 "$api_pid" 2>/dev/null; then
  wait "$api_pid"
  exit_code=$?
else
  wait "$nginx_pid"
  exit_code=$?
fi

shutdown
exit "$exit_code"
