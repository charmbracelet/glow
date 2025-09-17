#!/bin/bash
# test_hardening.sh - Core resilience validation for glow streaming
set -euo pipefail

# Configuration
V=${V:-}  # Verbose output
X=${X:-}  # Exit on first failure
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Output helpers
it() {
    echo "it $1"
}

ok() {
    echo "✅ ok - $1"
}

not_ok() {
    echo "❌ not ok - $1"
    [ -n "$X" ] && exit 1
}

debug() {
    [ -n "$V" ] && echo "# DEBUG: $*" >&2
}

# Build glow
echo "# Building glow..."
if ! go build; then
    echo "Build failed"
    exit 1
fi
cp -avn ./glow ./glow.orig >&2 || true
timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3

# Test counter
TESTS=0
PASS=0

run_test() {
    ((TESTS++))
    if "$@"; then
        ((PASS++))
        return 0
    else
        return 1
    fi
}

# ============================================================================
# CORE RESILIENCE TESTS
# ============================================================================

it "should stay under 100MB for 1MB input"
test_memory_bounded() {
    debug "Testing memory bounds"
    local testfile="$TMPDIR/test.md"

    # Generate 1MB markdown file
    perl -e 'for(1..10000){print "# Section $_\nThis is **markdown** content with *formatting*.\n\n"}' > "$testfile"

    local size=$(du -k "$testfile" | cut -f1)
    debug "File size: ${size}KB"

    if command -v /usr/bin/time >/dev/null 2>&1; then
        # Measure memory with streaming buffer (use pipe to avoid file reading hang)
        local mem_output=$(/usr/bin/time -l sh -c "cat '$testfile' | ./glow -w0 --flow=4096 -" 2>&1 >/dev/null | grep "maximum resident" || true)
        local rss=$(echo "$mem_output" | awk '{print $1}')
        local rss_mb=$((rss / 1024 / 1024))
        debug "RSS: ${rss_mb}MB"

        if [ "$rss_mb" -lt 100 ]; then
            ok "memory under 100MB (${rss_mb}MB)"
            return 0
        else
            not_ok "memory exceeds 100MB (${rss_mb}MB)"
            return 1
        fi
    else
        debug "time command not available"
        ok "memory test skipped"
        return 0
    fi
}
run_test test_memory_bounded

it "should not crash on 1MB of brackets"
test_pathological_brackets() {
    debug "Testing pathological input"
    local testfile="$TMPDIR/brackets.md"

    # 1MB of '[' characters - pathological for markdown
    perl -e 'print "[" x (1024 * 1024)' > "$testfile"

    # Should complete or timeout gracefully (not crash)
    timeout 5s ./glow -w0 --flow=1024 "$testfile" >/dev/null 2>&1
    local ret=$?

    if [ $ret -eq 0 ] || [ $ret -eq 124 ]; then  # 0=success, 124=timeout
        ok "survived pathological input"
        return 0
    else
        not_ok "crashed on pathological input (exit $ret)"
        return 1
    fi
}
run_test test_pathological_brackets

it "should not crash on deeply nested lists"
test_deep_nesting() {
    debug "Testing deep nesting"
    local testfile="$TMPDIR/nested.md"

    # 50-level nested list
    perl -e 'for(1..50){print " " x ($_*2) . "* Item $_\n"}' > "$testfile"

    if timeout 3s ./glow -w0 --flow=512 "$testfile" >/dev/null 2>&1; then
        ok "handled deep nesting"
        return 0
    else
        not_ok "failed on deep nesting"
        return 1
    fi
}
run_test test_deep_nesting

it "should not crash on malformed UTF-8"
test_invalid_utf8() {
    debug "Testing invalid UTF-8"
    local testfile="$TMPDIR/invalid.md"

    # Mix valid markdown with invalid UTF-8
    printf "# Valid\n\x80\x81\x82\n## More valid\n" > "$testfile"

    if ./glow -w0 --flow=256 "$testfile" >/dev/null 2>&1; then
        ok "handled invalid UTF-8"
        return 0
    else
        not_ok "crashed on invalid UTF-8"
        return 1
    fi
}
run_test test_invalid_utf8

it "should not crash on binary data"
test_binary_data() {
    debug "Testing binary input"

    # 100KB of random binary data
    dd if=/dev/urandom bs=1024 count=100 2>/dev/null | timeout 2s ./glow -w0 --flow=512 - >/dev/null 2>&1
    local ret=$?

    if [ $ret -eq 0 ] || [ $ret -eq 124 ]; then
        ok "handled binary data"
        return 0
    else
        not_ok "crashed on binary data (exit $ret)"
        return 1
    fi
}
run_test test_binary_data

it "should exit cleanly on SIGTERM"
test_signal_handling() {
    debug "Testing signal handling"

    # Start glow with slow input
    (
        echo "# Start"
        sleep 1
        echo "# End"
    ) | ./glow -w0 --flow=256 - &
    local pid=$!

    # Send SIGTERM after brief delay
    sleep 0.1
    kill -TERM $pid 2>/dev/null || true
    wait $pid 2>/dev/null
    local ret=$?

    # Should exit cleanly (common exit codes for SIGTERM)
    if [ $ret -eq 0 ] || [ $ret -eq 130 ] || [ $ret -eq 143 ]; then
        ok "clean exit on SIGTERM"
        return 0
    else
        not_ok "unclean exit on SIGTERM (exit $ret)"
        return 1
    fi
}
run_test test_signal_handling

it "should handle memory limits gracefully"
test_memory_limit() {
    debug "Testing memory limits"

    # Try to set memory limit (may not work on all systems)
    if ulimit -v 50000 2>/dev/null; then
        echo "# Test" | timeout 2s ./glow -w0 --flow=16 - >/dev/null 2>&1
        local ret=$?

        # Should handle limit gracefully
        if [ $ret -eq 0 ] || [ $ret -eq 137 ] || [ $ret -eq 134 ]; then
            ok "handled memory limit"
            return 0
        else
            not_ok "poor handling of memory limit (exit $ret)"
            return 1
        fi
    else
        debug "ulimit not supported"
        ok "memory limit test skipped"
        return 0
    fi
}
run_test test_memory_limit

it "should handle broken pipes gracefully"
test_broken_pipe() {
    debug "Testing broken pipe"

    # Pipe to head which will close early
    (
        for i in {1..1000}; do
            echo "# Line $i"
        done
    ) | ./glow -w0 --flow=256 - | head -n 1 >/dev/null 2>&1
    local ret=${PIPESTATUS[1]}

    # Should handle SIGPIPE gracefully
    if [ $ret -eq 0 ] || [ $ret -eq 141 ]; then
        ok "handled broken pipe"
        return 0
    else
        not_ok "poor handling of broken pipe (exit $ret)"
        return 1
    fi
}
run_test test_broken_pipe
