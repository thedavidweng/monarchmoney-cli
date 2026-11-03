#!/usr/bin/env bash
set -euo pipefail

key_file=${1:?key file required}
directory=${2:?keyring directory required}
fingerprint=${3:?fingerprint required}
if [[ ! "$fingerprint" =~ ^[A-F0-9]{40}$ ]]; then
  echo 'invalid signing key fingerprint' >&2
  exit 1
fi

mkdir -m 700 "$directory" "$directory/staging" "$directory/trusted"
gpg --homedir "$directory/staging" --batch --import "$key_file"
gpg --homedir "$directory/staging" --batch --export "$fingerprint" > "$directory/pinned.gpg"
if [[ ! -s "$directory/pinned.gpg" ]]; then
  echo 'pinned signing key not found' >&2
  exit 1
fi
gpg --homedir "$directory/trusted" --batch --import "$directory/pinned.gpg"
printf 'trusted-key %s\nimport-options merge-only\nno-auto-key-import\nno-auto-key-retrieve\n' \
  "$fingerprint" > "$directory/trusted/gpg.conf"
