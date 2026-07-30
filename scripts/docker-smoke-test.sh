#!/usr/bin/env bash
# Builds the Docker image and runs it as a real container to verify it works
# end to end: this is not exercised by `go test` since it adds no Go code,
# only packaging. Requires a working `docker` on the machine running it.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
image_tag="batocera-scrap-manager:smoke-test"
container_name="batocera-scrap-manager-smoke-test"
host_port=18080
base_url="http://localhost:${host_port}"

workdir="$(mktemp -d)"
cleanup() {
  docker rm -f "$container_name" >/dev/null 2>&1 || true
  rm -rf "$workdir"
}
trap cleanup EXIT

wait_for_http() {
  for _ in $(seq 1 40); do
    if curl -fsS -o /dev/null "$base_url/"; then
      return 0
    fi
    sleep 0.5
  done
  echo "server never answered on $base_url" >&2
  docker logs "$container_name" >&2 || true
  return 1
}

echo "==> building image"
docker build -t "$image_tag" "$repo_root"

echo "==> scenario 1: container started with only an empty registry folder configured"
# 'serve' refuses to start with no registry folder configured at all (see
# internal/cli/common.go), so the minimal case a container can serve is a
# config file naming an (empty) registry folder — not a fully absent config.
mkdir -p "$workdir/empty-registry"
cat >"$workdir/empty-config.json" <<'JSON'
{
  "registry_folder": "/data/registry"
}
JSON
docker run -d --name "$container_name" \
  -p "${host_port}:8080" \
  -e BATOCERA_SCRAP_MANAGER_CONFIG=/data/config.json \
  -v "$workdir/empty-registry:/data/registry" \
  -v "$workdir/empty-config.json:/data/config.json:ro" \
  "$image_tag" >/dev/null
wait_for_http
body="$(curl -fsS "$base_url/")"
if ! grep -q "No games in the registry yet." <<<"$body"; then
  echo "expected the empty-registry message on first run, got:" >&2
  echo "$body" >&2
  exit 1
fi
docker rm -f "$container_name" >/dev/null

echo "==> scenario 2: registry, ROMs folder and config file mounted from the host"
mkdir -p "$workdir/registry/nes" "$workdir/roms/nes"
cat >"$workdir/registry/nes/smoketestgame.json" <<'JSON'
{
  "path": "./smoketestgame.nes",
  "name": "Smoke Test Game",
  "desc": "Placeholder description so the entry is kept."
}
JSON
cat >"$workdir/config.json" <<'JSON'
{
  "registry_folder": "/data/registry",
  "roms_folders": ["/data/roms/nes"]
}
JSON

docker run -d --name "$container_name" \
  -p "${host_port}:8080" \
  -e BATOCERA_SCRAP_MANAGER_CONFIG=/data/config.json \
  -v "$workdir/registry:/data/registry" \
  -v "$workdir/roms:/data/roms" \
  -v "$workdir/config.json:/data/config.json:ro" \
  "$image_tag" >/dev/null
wait_for_http
body="$(curl -fsS "$base_url/")"
if ! grep -q 'systems__name">nes<' <<<"$body"; then
  echo "expected the mounted registry's 'nes' system to be listed, got:" >&2
  echo "$body" >&2
  exit 1
fi

echo "==> scenario 3: the container shuts down cleanly on SIGTERM"
docker stop --time 15 "$container_name" >/dev/null
status="$(docker inspect -f '{{.State.Status}}' "$container_name")"
exit_code="$(docker inspect -f '{{.State.ExitCode}}' "$container_name")"
if [ "$status" != "exited" ] || [ "$exit_code" != "0" ]; then
  echo "expected the container to exit cleanly (code 0), got status=$status exit_code=$exit_code" >&2
  exit 1
fi

echo "OK: all Docker smoke scenarios passed"
