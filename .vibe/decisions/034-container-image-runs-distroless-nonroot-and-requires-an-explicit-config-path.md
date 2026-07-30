---
date: 2026-07-30
status: accepted
---
# Container image runs distroless nonroot, and requires an explicit config path

**Context:** Adding a Dockerfile so `serve` can run in a container without a local Go toolchain. The binary is a statically-built (`CGO_ENABLED=0`) Go program with no libc dependency, so the runtime stage's base image and default user were open choices, as was how the container resolves its configuration file path.

**Decision:** The runtime stage uses `gcr.io/distroless/static-debian12:nonroot` and runs as its built-in non-root user (uid 65532). The entrypoint is written in exec form (`ENTRYPOINT ["/batocera-scrap-manager", "serve", ...]`), not shell form. Container usage is documented as requiring `BATOCERA_SCRAP_MANAGER_CONFIG` to be set explicitly to a mounted path, rather than relying on `serve`'s default OS-user-config-dir resolution.

**Reason:** A static Go binary needs neither a shell nor a package manager at runtime, so distroless removes CVE surface that alpine's musl + shell + package manager would carry, while still providing CA certificates and a non-root user out of the box (unlike raw `scratch`). Running as non-root matters concretely here: `config.Save` creates directories and writes files under the configured path, and the container user must own whatever host directory is bind-mounted for the config, registry, and ROMs folders, or writes fail. Exec form makes the binary PID 1 so it receives `SIGTERM` directly for the graceful shutdown `serve` already implements — shell form would wrap it in `/bin/sh -c`, which distroless doesn't even have. `os.UserConfigDir()`'s default resolution depends on `$HOME`, which is unset or non-writable for the nonroot distroless user, so leaving container users to rely on the default would surface as a confusing failure deep in a request handler rather than at startup.

**Rejected alternatives:** `alpine` as the runtime base (more CVE surface for no benefit, since the binary needs no shell or package manager at runtime). `scratch` (no CA certs, no built-in non-root user, no tzdata). Running as root (simpler bind-mount permissions, but violates least privilege for a service that writes to host-mounted folders). Relying on `serve`'s default config path resolution inside the container (fragile given distroless's `$HOME` situation).
