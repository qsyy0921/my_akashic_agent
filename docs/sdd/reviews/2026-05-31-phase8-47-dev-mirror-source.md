# Review: development mirror source

Spec:
`docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Confirmed local Go uses `GOPROXY=https://goproxy.cn,direct` and `GOSUMDB=off`.
- Added Docker Desktop user-level `registry-mirrors` fallback in
  `C:\Users\qsyy0921\.docker\daemon.json` with a timestamped backup beside the
  original file.
- Verified the Docker mirror path by pulling
  `docker.m.daocloud.io/library/nats:2-alpine`.
- Documented that Docker Desktop must be restarted before `docker info` reports
  the new daemon mirror list.

Tests run:
- `go env GOPROXY GOSUMDB`
- `docker pull docker.m.daocloud.io/library/nats:2-alpine`
- `docker info --format '{{json .RegistryConfig.Mirrors}}'`

Findings:
- The NATS image already existed locally, so current MQ smoke does not depend
  on Docker Hub.
- Docker daemon mirror reporting still returns `null` until Docker Desktop is
  restarted; this is expected for a running daemon.
- Avoid setting a permanent Docker daemon proxy to `127.0.0.1:7897` for now,
  because pulls would become dependent on that local proxy always being alive.

Decision:
Keep Go module mirror and Docker registry mirror as development environment
fallbacks. Continue using per-command `HTTP_PROXY` / `HTTPS_PROXY` for Git or
external network operations that need the local `7897` proxy.

Follow-ups:
- Restart Docker Desktop before the next Docker image pull smoke if the daemon
  needs to use the configured registry mirrors automatically.
- Keep runtime local calls under `NO_PROXY=127.0.0.1,localhost`.
