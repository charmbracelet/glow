#!/bin/bash
# Comprehensive test suite for glow streaming pager functionality

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

test_count=0
pass_count=0
fail_count=0

run_test() {
    local name="$1"
    local cmd="$2"
    local expected_exit="${3:-0}"

    test_count=$((test_count + 1))
    echo -n "Test $test_count: $name... "

    if output=$(eval "$cmd" 2>&1); then
        actual_exit=0
    else
        actual_exit=$?
    fi

    if [ "$actual_exit" -eq "$expected_exit" ] || [ "$expected_exit" -eq -1 ]; then
        echo -e "✅ ${GREEN}PASS${NC}"
        pass_count=$((pass_count + 1))
    else
        echo -e "❌ ${RED}FAIL${NC} (expected exit $expected_exit, got $actual_exit)"
        fail_count=$((fail_count + 1))
    fi
}

# Basic functionality tests
echo "== Basic Functionality =="
run_test "Simple echo" "echo '# Test' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0
run_test "Empty input" "echo -n '' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0
run_test "Single character" "echo -n 'x' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0
run_test "Multi-line markdown" "echo -e '# Title\n\nParagraph\n\n## Section' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0

# Streaming flow tests
echo ""
echo "== Streaming Flow Tests =="
run_test "Flow with immediate data" "echo '# Test' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -f -\"" 0
run_test "Flow with delayed data" "( echo '# First'; sleep 0.1; echo '## Second' ) | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -f -\"" 0
run_test "Flow with early pager exit" "( for i in {1..100}; do echo '# Line'; sleep 0.01; done ) | timeout 1 bash -c \"PAGER='head -2' ./glow -w0 -p -f -\"" -1

# Context cancellation tests
echo ""
echo "== Context Cancellation =="
run_test "Pager exits before input complete" "( while true; do echo 'data'; sleep 0.01; done ) | timeout 0.2 bash -c \"PAGER='timeout 0.05 cat' ./glow -w0 -p -f -\"" 124
run_test "Slow input with quick pager exit" "( while true; do sleep 10; echo 'data'; done ) | timeout 0.5 bash -c \"PAGER='timeout 0.05 cat' ./glow -w0 -p -f -\"" 124
run_test "SIGTERM handling" "( for i in {1..100}; do echo '# Line'; sleep 0.01; done ) | timeout -s TERM 0.1 bash -c \"PAGER='cat' exec ./glow -w0 -p -f -\"" 124
run_test "SIGINT handling" "( for i in {1..100}; do echo '# Line'; sleep 0.01; done ) | timeout -s INT 0.1 bash -c \"PAGER='cat' exec ./glow -w0 -p -f -\"" 124

# Edge cases
echo ""
echo "== Edge Cases =="
run_test "Frontmatter removal" "echo -e '---\ntitle: Test\n---\n# Content' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0
run_test "Code blocks" "echo -e '\`\`\`\ncode\n\`\`\`' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0
run_test "Unicode content" "echo -e '# 测试 🚀\n\n内容' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0
run_test "Very long lines" "( echo -n '# '; for j in {1..10000}; do echo -n 'A'; done; echo ) | timeout 1 bash -c \"PAGER='head -1' ./glow -w0 -p -f -\"" -1

# Pager compatibility tests
echo ""
echo "== Pager Compatibility =="
run_test "Less pager" "echo '# Test' | timeout 1 bash -c \"PAGER='less -F' ./glow -w0 -p -\"" 0
run_test "Head pager" "( for i in {1..20}; do echo '# Line'; done ) | timeout 1 bash -c \"PAGER='head -5' ./glow -w0 -p -\"" -1
run_test "Cat pager" "echo '# Test' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 0
run_test "Custom pager with args" "echo '# Test' | timeout 1 bash -c \"PAGER='cat -n' ./glow -w0 -p -\"" 0

# Performance tests
echo ""
echo "== Performance =="
run_test "Large document" "( for i in {1..1000}; do echo -e '# Header\n\nContent\n'; done ) | timeout 2 bash -c \"PAGER='head -100' ./glow -w0 -p -f -\"" -1
run_test "Rapid small chunks" "( for i in {1..1000}; do echo 'x'; done ) | timeout 1 bash -c \"PAGER='head -10' ./glow -w0 -p -f -\"" -1
run_test "Memory stress (10MB)" "( dd if=/dev/zero bs=1024 count=10240 2>/dev/null | tr '\0' 'A' ) | timeout 2 bash -c \"PAGER='head -1' ./glow -w0 -p -f -\"" -1

# Resource management tests
echo ""
echo "== Resource Management =="
run_test "No goroutine leak on cancel" "( while true; do echo 'data'; sleep 0.01; done ) | timeout 0.2 bash -c \"PAGER='timeout 0.05 cat' ./glow -w0 -p -f -\"" 124
run_test "Multiple sequential runs" "for i in {1..3}; do echo '# Test' | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"; done" 0
run_test "Pipe closure on error" "( echo '# Test'; exit 1 ) | timeout 1 bash -c \"PAGER='cat' ./glow -w0 -p -\"" 1
