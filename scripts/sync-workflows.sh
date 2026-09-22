#!/usr/bin/env bash
# sync-workflows.sh: Verify and sync common workflows across sibling repositories.
set -euo pipefail

DEV_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CURRENT_REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Managed sibling repositories
REPOS=(
  "canvas-cli"
  "flickr-cli"
  "monarchmoney-cli"
  "money"
  "qualtrics-cli"
  "zenodo-cli"
  "tg-drive-cli"
)

# Common workflows to keep identical
SYNC_WORKFLOWS=(
  "ci.yml"
  "codeql.yml"
  "dependabot-automerge.yml"
  "release.yml"
)

MODE="sync"
if [ "${1:-}" = "--check" ]; then
  MODE="check"
fi

SOURCE_DIR="${CURRENT_REPO}/.github/workflows"

if [ "$MODE" = "check" ]; then
  echo "Checking workflow synchronization across repositories..."
  mismatches=0
  for repo in "${REPOS[@]}"; do
    target_dir="${DEV_ROOT}/${repo}/.github/workflows"
    if [ ! -d "$target_dir" ]; then
      continue
    fi
    for wf in "${SYNC_WORKFLOWS[@]}"; do
      # tg-drive-cli has customized flags for release.yml (WASM/parallelism)
      if { [ "$repo" = "tg-drive-cli" ] || [ "${CURRENT_REPO##*/}" = "tg-drive-cli" ]; } && [ "$wf" = "release.yml" ]; then
        continue
      fi
      # tg-drive-cli and qualtrics-cli have customized CI pipelines
      if { [ "$repo" = "tg-drive-cli" ] || [ "$repo" = "qualtrics-cli" ] || [ "${CURRENT_REPO##*/}" = "tg-drive-cli" ] || [ "${CURRENT_REPO##*/}" = "qualtrics-cli" ]; } && [ "$wf" = "ci.yml" ]; then
        continue
      fi
      # Only verify codeql.yml if it exists in target or source
      if [ "$wf" = "codeql.yml" ] && { [ ! -f "${target_dir}/${wf}" ] || [ ! -f "${SOURCE_DIR}/${wf}" ]; }; then
        continue
      fi
      if [ -f "${SOURCE_DIR}/${wf}" ] && [ -f "${target_dir}/${wf}" ]; then
        if ! cmp -s "${SOURCE_DIR}/${wf}" "${target_dir}/${wf}"; then
          echo "✗ Mismatch in ${repo}: ${wf}"
          mismatches=$((mismatches + 1))
        fi
      fi
    done
  done
  if [ "$mismatches" -gt 0 ]; then
    echo "Found ${mismatches} workflow mismatch(es). Run without --check to synchronize."
    exit 1
  fi
  echo "✓ All managed workflows are in sync!"
  exit 0
fi

echo "Synchronizing workflows from ${CURRENT_REPO##*/} to sibling repositories..."
for repo in "${REPOS[@]}"; do
  target_dir="${DEV_ROOT}/${repo}/.github/workflows"
  if [ ! -d "$target_dir" ] || [ "${DEV_ROOT}/${repo}" = "$CURRENT_REPO" ]; then
    continue
  fi
  for wf in "${SYNC_WORKFLOWS[@]}"; do
    if [ "$repo" = "tg-drive-cli" ] && [ "$wf" = "release.yml" ]; then
      continue
    fi
    if { [ "$repo" = "tg-drive-cli" ] || [ "$repo" = "qualtrics-cli" ]; } && [ "$wf" = "ci.yml" ]; then
      continue
    fi
    if [ "$wf" = "codeql.yml" ] && [ ! -f "${target_dir}/${wf}" ]; then
      continue
    fi
    if [ -f "${SOURCE_DIR}/${wf}" ]; then
      cp "${SOURCE_DIR}/${wf}" "${target_dir}/${wf}"
      echo "✓ Synced ${wf} -> ${repo}"
    fi
  done
  if [ -f "${CURRENT_REPO}/scripts/sync-workflows.sh" ] && [ -f "${DEV_ROOT}/${repo}/scripts/sync-workflows.sh" ]; then
    cp "${CURRENT_REPO}/scripts/sync-workflows.sh" "${DEV_ROOT}/${repo}/scripts/sync-workflows.sh"
    echo "✓ Synced sync-workflows.sh -> ${repo}"
  fi
done

echo "✓ Workflows successfully synchronized across all repositories!"
