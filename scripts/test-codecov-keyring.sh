#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -m 700 "$work/signer" "$work/other"

for home in signer other; do
  gpg --homedir "$work/$home" --batch --pinentry-mode loopback --passphrase '' \
    --quick-generate-key "$home test" ed25519 sign 1d
done
fingerprint=$(gpg --homedir "$work/signer" --batch --with-colons --list-keys | awk -F: '$1 == "fpr" { print $10; exit }')
gpg --homedir "$work/signer" --batch --export > "$work/signer.gpg"
gpg --homedir "$work/other" --batch --export > "$work/other.gpg"
cat "$work/signer.gpg" "$work/other.gpg" > "$work/mixed.gpg"

if bash "$root/scripts/prepare-codecov-keyring.sh" "$work/other.gpg" "$work/missing" "$fingerprint" >/dev/null 2>&1; then
  echo 'accepted a key file without the expected signer' >&2
  exit 1
fi
bash "$root/scripts/prepare-codecov-keyring.sh" "$work/mixed.gpg" "$work/prepared" "$fingerprint"
verify_home="$work/prepared/trusted"

printf 'checksums\n' > "$work/sums"
for home in signer other; do
  gpg --homedir "$work/$home" --batch --pinentry-mode loopback --passphrase '' \
    --detach-sign --output "$work/$home.sig" "$work/sums"
done
gpg --homedir "$verify_home" --batch --no-default-keyring --import "$work/mixed.gpg"
gpg --homedir "$verify_home" --batch --verify "$work/signer.sig" "$work/sums"
if gpg --homedir "$verify_home" --batch --verify "$work/other.sig" "$work/sums" >/dev/null 2>&1; then
  echo 'accepted a signature from another key' >&2
  exit 1
fi
printf 'tampered\n' >> "$work/sums"
if gpg --homedir "$verify_home" --batch --verify "$work/signer.sig" "$work/sums" >/dev/null 2>&1; then
  echo 'accepted altered content' >&2
  exit 1
fi
echo 'pinned signature verified; missing signer, other signer, and altered content rejected'
