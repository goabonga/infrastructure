#!/usr/bin/env bash

# SPDX-License-Identifier: MIT
# Copyright (c) 2026 Chris <goabonga@pm.me>

# Rewrite the current HEAD commit with a Conventional Commits prefix inferred
# from the files it touches, then amend with --reset-author and a GPG signature
# so the commit is attributed to and signed by the maintainer's identity
# (configured by the caller workflow).
#
# Routing rules (first-match wins):
#
#   .github/workflows/*.yml   -> ci:
#   go.mod / go.sum           -> chore(deps):
#   www/package*.json         -> chore(deps): (frontend)
#   anything else             -> chore(deps):
#
# This repository is a single Go module, so dependency bumps land in go.mod /
# go.sum and map to chore(deps) rather than a per-component scope. Designed to
# be invoked by `git rebase --exec` against each commit of a Dependabot pull
# request. Stand-alone use is fine too.

set -euo pipefail

changed=$(git show --name-only --pretty='' HEAD)

case "$changed" in
    *.github/workflows/*)
        prefix="ci"
        ;;
    *go.mod* | *go.sum*)
        prefix="chore(deps)"
        ;;
    *www/package*.json*)
        prefix="chore(deps)"
        ;;
    *)
        prefix="chore(deps)"
        ;;
esac

original=$(git log -1 --pretty=%B HEAD)

# Strip any existing Conventional-style prefix from the subject and lowercase
# the leading "Bump" that Dependabot uses by default.
subject=$(printf '%s' "$original" \
    | head -n 1 \
    | sed -E 's/^[a-z-]+(\([^)]+\))?:[[:space:]]*//' \
    | sed 's/^[Bb]ump /bump /')

# Preserve the body but drop any `Co-Authored-By:` trailer - this repo enforces
# a strict no-co-author policy.
body=$(printf '%s' "$original" \
    | tail -n +2 \
    | sed '/^Co-Authored-By:/d')

if [ -n "$body" ]; then
    new_msg=$(printf '%s: %s\n\n%s' "$prefix" "$subject" "$body")
else
    new_msg=$(printf '%s: %s' "$prefix" "$subject")
fi

git commit --amend --reset-author -m "$new_msg" --quiet
