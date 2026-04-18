#!/bin/sh

# These knobs allow slow environments to tune readiness polling.
PROXY_HOST="${PROXY_HOST:-127.0.0.1}"
PROXY_PORT="${PROXY_PORT:-8080}"
PROXY_START_TIMEOUT_SECONDS="${PROXY_START_TIMEOUT_SECONDS:-30}"
PROXY_START_CHECK_INTERVAL_SECONDS="${PROXY_START_CHECK_INTERVAL_SECONDS:-1}"

wait_for_proxy() {
	# Readiness probing depends on netcat; fail fast if unavailable.
	if ! command -v nc >/dev/null 2>&1; then
		echo "[entrypoint] nc is required for proxy readiness checks"
		kill "$proxy_pid" 2>/dev/null || true
		wait "$proxy_pid" 2>/dev/null || true
		exit 1
	fi

	start_ts="$(date +%s)"

	while :; do
		# Surface proxy startup failures instead of waiting for timeout.
		if ! kill -0 "$proxy_pid" 2>/dev/null; then
			wait "$proxy_pid"
			proxy_status="$?"
			echo "[entrypoint] proxy exited before readiness check completed (code=${proxy_status})"
			exit "$proxy_status"
		fi

		# TCP check avoids calling /logs and producing noisy handler errors.
		if nc -z "$PROXY_HOST" "$PROXY_PORT" >/dev/null 2>&1; then
			echo "[entrypoint] proxy is ready on ${PROXY_HOST}:${PROXY_PORT}"
			return 0
		fi

		now_ts="$(date +%s)"
		elapsed="$((now_ts - start_ts))"

		if [ "$elapsed" -ge "$PROXY_START_TIMEOUT_SECONDS" ]; then
			echo "[entrypoint] timeout waiting for proxy readiness on ${PROXY_HOST}:${PROXY_PORT}"
			# Prevent leaving a zombie proxy process in failed startup path.
			kill "$proxy_pid" 2>/dev/null || true
			wait "$proxy_pid" 2>/dev/null || true
			exit 1
		fi

		sleep "$PROXY_START_CHECK_INTERVAL_SECONDS"
	done
}

/go/bin/proxy &
proxy_pid="$!"

# Start vector only after proxy is actually listening.
echo "[entrypoint] waiting for proxy readiness..."
wait_for_proxy

vector "$@" &

# Keep container alive while both are running; exit on first failure/stop.
wait -n

exit $?
