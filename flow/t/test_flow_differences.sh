#!/bin/bash

# Test that flow modes produce acceptable differences (reference links only)

set -euo pipefail

echo "🔍 Testing flow mode reference link resolution differences..."

# Build glow
go build .
cp -avn ./glow ./glow.orig >&2 || true
timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3

# Use the fixed README outputs
(
    glow.orig () { while read -r; do echo "$REPLY"; done | ./glow.orig "$@"; }
    glow () { while read -r; do echo "$REPLY"; done | ./glow -f "$@"; }
    echo -en "./glow.orig\t \t"; glow.orig -w0 < README.md | tee /tmp/test_flow_orig.md | md5sum | cut -d' ' -f1
    for flow in -1 0 1 16 256 4096 65536 1048576; do
        echo -en "./glow\t--flow=$flow\t"; ./glow -w0 -f=$flow < README.md | tee /tmp/test_flow_windowed$flow.md | md5sum | cut -d' ' -f1
    done
    cp -a /tmp/test_flow_windowed-1.md /tmp/test_flow_unbuffered.md
    cp -a /tmp/test_flow_windowed1.md /tmp/test_flow_windowed.md
) \
| column -s$'\t' -t

failed=0
for flow in -1 0 1 16 256 4096 65536 1048576; do
    if command diff -q /tmp/test_flow_orig.md /tmp/test_flow_windowed$flow.md; then
        echo "✅ PASS: flow=$flow matches"
    elif command diff -u <(grep -Ev 'contribute|releases' /tmp/test_flow_orig.md) <(grep -Ev 'contribute|releases' /tmp/test_flow_windowed$flow.md); then
        echo "✅ PASS: flow=$flow differs (reference links)"
    else
        command diff -q /tmp/test_flow_orig.md /tmp/test_flow_windowed$flow.md || true
        echo "❌ FAIL: flow=$flow differs"
        ((++failed <= ${GLOW_TEST_MAX_FAILED:-failed})) || break
    fi
done

! ((failed))
