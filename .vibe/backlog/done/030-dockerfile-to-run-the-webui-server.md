---
status: done
---
# Dockerfile to Run the WebUI Server

## Description
Add a `Dockerfile` at the repository root that builds a container image running `batocera-scrap-manager serve`, so the web UI can be started without a local Go toolchain. The image must expose the server's listening port and let the registry folder, ROMs folders, and config file be supplied from outside the container (bind mounts / environment variables), since `serve` reads its configuration from `BATOCERA_SCRAP_MANAGER_CONFIG` (or the OS user config dir) and serves the registry and ROMs folders referenced by that configuration.

## Acceptance Criteria
- [ ] `docker build .` produces an image that runs `batocera-scrap-manager serve` as its entrypoint/default command
- [ ] The container exposes the port `serve` listens on (default `8080`, overridable via `--addr`)
- [ ] The registry folder, ROMs folders, and config file can be mounted into the container from the host (e.g. via volumes) and are picked up by the running server
- [ ] Running the built image and requesting the exposed port returns the web UI's game list page

## Notes
Use a multi-stage build (Go build stage + a minimal runtime base) to keep the final image small. `serve` binds to `0.0.0.0:8080` by default, which is container-friendly. The config file location is controlled by the `BATOCERA_SCRAP_MANAGER_CONFIG` environment variable (see `internal/config/config.go`) — the Dockerfile/README should document setting it to a mounted path, or mounting a pre-populated config file at the default OS user config dir location.
