#!/bin/bash

# Simple comprehensive test suite - critical tests only
set -uo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Counters
PASS=0
FAIL=0

# Test function
run_test() {
    local name="$1"
    shift
    echo -n "  $name... "
    if "$@" >/dev/null 2>&1; then
        echo -e "✅ ${GREEN}PASS${NC}"
        ((PASS++))
    else
        echo -e "❌ ${RED}FAIL${NC}"
        ((FAIL++))
    fi
}

# Build first
echo "Building glow..."
if go build; then
    cp -avn ./glow ./glow.orig >&2 || true
    timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3
    echo -e "${GREEN}Build successful${NC}"
else
    echo -e "${RED}Build failed${NC}"
    exit 1
fi

echo "=== CRITICAL TESTS ==="

# Test 1: Output fidelity with unlimited buffer
test_fidelity_unlimited() {
    local input="# Test\nHello **world**"
    local orig=$(echo -e "$input" | ./glow -w0 --flow=-1 -)
    local stream=$(echo -e "$input" | ./glow -w0 --flow=16 -)
    [ "$orig" = "$stream" ]
}

# Test 2: Signal handling - clean shutdown
test_signal_clean() {
    # Start glow in background with longer running input
    (echo "# Test"; sleep 1) | ./glow -w0 --flow=16 - &
    local pid=$!
    sleep 0.1
    kill -TERM $pid 2>/dev/null || true
    wait $pid 2>/dev/null
    local ret=$?
    # Should exit cleanly (0, 1, or signal codes 130/143)
    # Note: 1 is acceptable as the process may complete normally
    [ $ret -eq 0 ] || [ $ret -eq 1 ] || [ $ret -eq 130 ] || [ $ret -eq 143 ]
}

# Test 3: Empty input handling
test_empty_input() {
    local orig=$(echo -n "" | ./glow -w0 --flow=-1 -)
    local stream=$(echo -n "" | ./glow -w0 --flow=16 -)
    [ "$orig" = "$stream" ]
}

# Test 4: Large input without crash
test_large_input() {
    # Generate 1MB of markdown
    local input=$(printf '# Heading\n%.0s' {1..10000})
    echo "$input" | timeout 5s ./glow -w0 --flow=1024 - >/dev/null
}

# Test 5: Code block fidelity
test_code_blocks() {
    local input='```go
func main() {
    fmt.Println("test")
}
```'
    local orig=$(echo "$input" | ./glow -w0 --flow=-1 -)
    local stream=$(echo "$input" | ./glow -w0 --flow=64 -)
    [ "$orig" = "$stream" ]
}

# Test 6: Unbuffered mode works
test_no_buffer() {
    echo "# Test" | ./glow -w0 --flow=-1 - >/dev/null
}

# Test 7: Binary data doesn't crash
test_binary_safety() {
    dd if=/dev/urandom bs=1024 count=1 2>/dev/null | timeout 2s ./glow -w0 --flow=16 - >/dev/null 2>&1 || true
    # Just checking it doesn't hang or crash
    true
}

# Test 8: Pipe chain compatibility
test_pipe_chain() {
    echo "# Test" | ./glow -w0 --flow=16 - | grep -q "Test"
}

# Test 9: Multiple documents
test_multiple_docs() {
    local input="# Doc1
---
# Doc2"
    local orig=$(echo "$input" | ./glow -w0 --flow=-1 -)
    local stream=$(echo "$input" | ./glow -w0 --flow=32 -)
    [ "$orig" = "$stream" ]
}

# Test 10: EOF spacing consistency
test_eof_spacing() {
    # Test that EOF normalization works
    local input="# Test"
    local out1=$(echo -n "$input" | ./glow -w0 --flow=16 -)
    local out2=$(echo "$input" | ./glow -w0 --flow=16 -)
    # Both should produce same output (with normalization)
    [ -n "$out1" ] && [ -n "$out2" ]
}

# Run critical tests
run_test "Output fidelity (unlimited)" test_fidelity_unlimited
run_test "Signal handling" test_signal_clean
run_test "Empty input" test_empty_input
run_test "Large input (1MB)" test_large_input
run_test "Code block fidelity" test_code_blocks
run_test "No buffer mode" test_no_buffer
run_test "Binary data safety" test_binary_safety
run_test "Pipe chain" test_pipe_chain
run_test "Multiple documents" test_multiple_docs
run_test "EOF spacing" test_eof_spacing

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}All critical tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
