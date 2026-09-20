#!/usr/bin/env bash
set -euo pipefail

python3 - <<'EOF'
import re, glob, sys

defined = {}
for f in sorted(glob.glob('queries/**/*.graphql', recursive=True)):
    for m in re.finditer(r'(?:query|mutation|subscription)\s+(\w+)', open(f).read()):
        op = m.group(1)
        if op in ('PayloadErrorFields',):
            continue
        defined.setdefault(op, []).append(f)

dups = {op: files for op, files in defined.items() if len(files) > 1}

go_sources = []
for f in sorted(glob.glob('internal/**/*.go', recursive=True) + glob.glob('cmd/**/*.go', recursive=True)):
    if f.endswith('_test.go'):
        continue
    go_sources.append((f, open(f).read()))

unreferenced = {}
for op, files in defined.items():
    needle = f'"{op}"'
    if not any(needle in src for _, src in go_sources):
        unreferenced[op] = files

dangling = {}
for f, src in go_sources:
    for m in re.finditer(r'OperationName:\s*"([^"]+)"', src):
        op = m.group(1)
        if op not in defined:
            dangling.setdefault(op, []).append(f)

failed = False
for op, files in sorted(dups.items()):
    print(f'duplicate operation {op} in: {files}')
    failed = True
for op, files in sorted(unreferenced.items()):
    print(f'unreferenced operation {op} defined in: {files}')
    failed = True
for op, files in sorted(dangling.items()):
    print(f'dangling OperationName {op} used in: {files}')
    failed = True

if failed:
    print('API drift check failed: fix the operations above or remove the dead queries.')
    sys.exit(1)
print(f'API drift check passed: {len(defined)} operations defined and referenced.')
EOF
