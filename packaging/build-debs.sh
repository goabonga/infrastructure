#!/usr/bin/env bash

# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Chris <goabonga@pm.me>

# Build Debian packages for the Go components.
#
# Usage:
#   packaging/build-debs.sh [VERSION] [component...]
#
# VERSION defaults to $VERSION or 0.0.0; ARCH defaults to $ARCH or amd64.
# With no components, every component is built. Output lands in dist/.

set -euo pipefail

VERSION="${1:-${VERSION:-0.0.0}}"
if [ "$#" -ge 1 ]; then shift; fi
ARCH="${ARCH:-amd64}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
MAINTAINER="Chris <goabonga@pm.me>"

# component -> "cmd-dir:binary:service-unit" (empty service = not a daemon)
declare -A COMPONENTS=(
  [infra]="cli:infra:"
  [infra-api]="api:infra-api:infra-api"
  [infra-agent]="agent:infra-agent:infra-agent"
  [infra-controller-manager]="controller-manager:infra-controller-manager:infra-controller-manager"
  [terraform-provider-infra]="provider:terraform-provider-infra:"
  [infra-exporter]="exporter:infra-exporter:infra-exporter"
  [infra-idp]="idp:infra-idp:infra-idp"
  [infra-container-init]="container-init:infra-container-init:"
)

build_deb() {
  local pkg="$1"
  if [ -z "${COMPONENTS[$pkg]:-}" ]; then
    echo "unknown component: $pkg" >&2
    return 1
  fi
  local dir bin svc
  IFS=: read -r dir bin svc <<<"${COMPONENTS[$pkg]}"

  local stage
  stage="$(mktemp -d)"
  trap 'rm -rf "$stage"' RETURN

  install -d "$stage/DEBIAN" "$stage/usr/local/bin"
  CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
    go build -ldflags='-s -w' -o "$stage/usr/local/bin/$bin" "$ROOT/cmd/$dir/"

  cat >"$stage/DEBIAN/control" <<EOF
Package: $pkg
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: $MAINTAINER
Description: Declarative Linux-native infrastructure: $pkg
EOF

  if [ -n "$svc" ]; then
    install -d "$stage/lib/systemd/system"
    install -m 0644 "$ROOT/deploy/systemd/$svc.service" "$stage/lib/systemd/system/$svc.service"
    cat >"$stage/DEBIAN/postinst" <<EOF
#!/bin/sh
set -e
systemctl daemon-reload || true
EOF
    cat >"$stage/DEBIAN/prerm" <<EOF
#!/bin/sh
set -e
if [ "\$1" = remove ]; then
  systemctl disable --now $svc.service || true
fi
EOF
    chmod 0755 "$stage/DEBIAN/postinst" "$stage/DEBIAN/prerm"
  fi

  install -d "$DIST"
  dpkg-deb --root-owner-group --build "$stage" "$DIST/${pkg}_${VERSION}_${ARCH}.deb"
}

targets=("$@")
if [ "${#targets[@]}" -eq 0 ]; then
  targets=("${!COMPONENTS[@]}")
fi
for pkg in "${targets[@]}"; do
  build_deb "$pkg"
done
echo "built: $(ls "$DIST"/*.deb 2>/dev/null | wc -l) package(s) in $DIST"
