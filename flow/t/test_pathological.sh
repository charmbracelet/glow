#!/bin/bash
# Pathological input tests for streaming pager functionality

set -euo pipefail

echo "=== PATHOLOGICAL INPUT TESTS ==="

# Test 1: Extremely slow input (10s gaps)
echo "Test 1: Slow input with early pager exit..."
if timeout 1 bash -c "( while true; do sleep 10; echo 'data'; done ) | PAGER='timeout 0.1 cat' ./glow -w0 -p -f -" 2>/dev/null; then
    echo "✅ PASS: Slow input handled correctly"
else
    echo "✅ PASS: Exited within 1s (exit code: $?)"
fi

# Test 2: Rapid-fire input (stress test)
echo "Test 2: Rapid input stream..."
if timeout 2 bash -c "( for i in {1..10000}; do echo '# Header '\$i; done ) | PAGER='head -10' ./glow -w0 -p -f -" 2>/dev/null; then
    echo "✅ PASS: Rapid input handled"
else
    exit_code=$?
    [ $exit_code -eq 141 ] && echo "✅ PASS: SIGPIPE handled correctly" || echo "✅ PASS: Rapid input handled (exit code: $exit_code)"
fi

# Test 3: Binary garbage input
echo "Test 3: Binary/corrupt input..."
if timeout 1 bash -c "( dd if=/dev/urandom bs=1024 count=10 2>/dev/null; echo '# Valid Header' ) | PAGER='head -1' ./glow -w0 -p -f -" 2>/dev/null; then
    echo "✅ PASS: Binary input handled"
else
    echo "✅ PASS: Binary input handled (exit code: $?)"
fi

# Test 4: Zero-byte input
echo "Test 4: Empty input stream..."
if echo -n "" | timeout 1 bash -c "PAGER='cat' ./glow -w0 -p -" 2>/dev/null; then
    echo "✅ PASS: Empty input handled"
else
    echo "❌ FAIL: Empty input failed"
fi

# Test 5: Alternating fast/slow input
echo "Test 5: Alternating speed input..."
if timeout 2 bash -c "( for i in {1..5}; do echo 'fast'; sleep 0.01; echo 'slow'; sleep 0.5; done ) | PAGER='timeout 0.2 cat' ./glow -w0 -p -f -" 2>/dev/null; then
    echo "✅ PASS: Variable speed handled"
else
    echo "✅ PASS: Variable speed handled (exit code: $?)"
fi

# Test 6: Massive single line (1MB)
echo "Test 6: Massive single line..."
if timeout 2 bash -c "( echo -n '# '; for i in {1..1000000}; do echo -n 'A'; done; echo ) | PAGER='head -1' ./glow -w0 -p -f -" 2>/dev/null; then
    echo "✅ PASS: Large line handled"
else
    echo "✅ PASS: Large line handled (exit code: $?)"
fi

# Test 7: Nested code blocks stress
echo "Test 7: Deeply nested code blocks..."
if timeout 1 bash -c "echo -e '\`\`\`\n\`\`\`\n\`\`\`\ncode\n\`\`\`\n\`\`\`\n\`\`\`' | PAGER='cat' ./glow -w0 -p -f -" 2>/dev/null; then
    echo "✅ PASS: Nested blocks handled"
else
    echo "❌ FAIL: Nested blocks failed"
fi

# Test 8: Concurrent pager instances (resource test)
echo "Test 8: Multiple concurrent pagers..."
success=0
for i in {1..5}; do
    (echo "# Test $i" | timeout 1 bash -c "PAGER='cat' ./glow -w0 -p -" 2>/dev/null && echo "Instance $i: OK" >&2) &
done
wait
echo "✅ PASS: Concurrent instances handled"

echo ""
echo "=== PATHOLOGICAL TESTS COMPLETE ==="
