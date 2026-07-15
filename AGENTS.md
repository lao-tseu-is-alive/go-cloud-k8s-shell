# Repository Notes for Codex

## Trivy mitigation context

- The Docker runtime image was migrated from Ubuntu to Alpine to remove OS-package CVEs.
- `kubectl` should come from Alpine `apk`, not from a manually downloaded upstream binary, so Trivy does not scan it as a separate vulnerable Go binary.
- `wget` was removed from the image; use `curl` in container helper scripts.
- `.dockerignore` should keep the build context limited to the Dockerfile inputs and avoid sending `cmd/server/front/node_modules`.

## x/crypto/autocert history

The previous remaining image finding was:

```text
home/gouser/go-shell-server
golang.org/x/crypto GO-2026-5932 UNKNOWN
```

This service does not import `golang.org/x/crypto/openpgp` directly. The dependency path is:

```text
go-cloud-k8s-shell/cmd/server
-> github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/gohttp
-> golang.org/x/crypto/acme/autocert
```

`go list -deps ./cmd/server` showed `golang.org/x/crypto/acme` and `golang.org/x/crypto/acme/autocert`, not `openpgp`.

## go-cloud-k8s-common v0.5.4 update

`github.com/lao-tseu-is-alive/go-cloud-k8s-common v0.5.4` moved autocert out of the default `pkg/gohttp` build path. This repository now requires `v0.5.4`, and `go mod tidy` removed the direct indirect `golang.org/x/crypto` requirement.

Verification notes:

- `go list -deps -f '{{if .Module}}{{.Module.Path}} {{.ImportPath}}{{end}}' ./cmd/server | rg 'golang.org/x/crypto|autocert|openpgp'` should return no output.
- The plain `go list -deps ./cmd/server | rg 'golang.org/x/crypto'` command may show `vendor/golang.org/x/crypto/...` packages from the Go toolchain's standard-library vendored tree; those are not the external module that Trivy previously reported.
- Run tests with dummy required env values:

```bash
JWT_AUTH_URL=/login \
JWT_CONTEXT_KEY=Testctx \
JWT_SECRET=testonlyjwtsecretvalue \
JWT_ISSUER_ID=testonlyjwtissuervalue \
ADMIN_PASSWORD='Testpass1!' \
ALLOWED_HOSTS=127.0.0.1,localhost \
LOG_FILE=DISCARD \
go test ./...
```

- Rebuild the image and rerun Trivy. Expected result: `0` OS findings and `0` gobinary findings.
