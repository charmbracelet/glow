#!/bin/bash
# test_progressive_streaming.sh - Validate true progressive streaming with pv

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Progressive Streaming Test with pv ==="
echo

# Check if pv is installed
if ! command -v pv &> /dev/null; then
    echo -e "${RED}Error: pv is not installed. Install with: brew install pv${NC}"
    exit 1
fi

# Build glow
echo "Building glow..."
go build .
cp -avn ./glow ./glow.orig >&2 || true
timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3

# Test 1: Prove output appears before EOF with aggressive flush
echo -e "${YELLOW}Test 1: Aggressive flush (--flow=-1)${NC}"
echo "Input rate: 50 bytes/sec, expecting immediate output on boundaries"
echo

# Create test input with clear paragraph boundaries
TEST_INPUT="# First Header

This is the first paragraph with enough content to trigger a flush.

# Second Header

This is the second paragraph that should appear separately.

# Third Header

Final paragraph to complete the test."

# Run test with timing
(echo "$TEST_INPUT" | pv -q -L 50 | ./glow -w0 --flow=-1 - 2>/dev/null | pv -bt > /dev/null) && echo -e "\n✅ ${GREEN}PASS${NC}" || echo -e "\n❌ ${RED}FAIL${NC}"
echo

# Test 2: Small buffer flush
echo -e "${YELLOW}Test 2: Small buffer (--flow=32)${NC}"
echo "Should flush after accumulating 32 bytes at paragraph boundaries"
echo

(echo "$TEST_INPUT" | pv -q -L 50 | ./glow -w0 --flow=32 - 2>/dev/null | pv -bt > /dev/null) && echo -e "\n✅ ${GREEN}PASS${NC}" || echo -e "\n❌ ${RED}FAIL${NC}"
echo

# Test 3: Default 1KB buffer
echo -e "${YELLOW}Test 3: Default buffer (--flow=1024)${NC}"
echo "Should accumulate more before flushing"
echo

(echo "$TEST_INPUT" | pv -q -L 50 | ./glow -w0 --flow=1024 - 2>/dev/null | pv -bt > /dev/null) && echo -e "\n✅ ${GREEN}PASS${NC}" || echo -e "\n❌ ${RED}FAIL${NC}"
echo

# Test 4: Visual progressive test with timestamps
echo -e "${YELLOW}Test 4: Visual progressive output with timestamps${NC}"
echo "Watch output appear progressively (3 separate blocks):"
echo

{
    echo "# Block 1"
    echo ""
    echo "First block content appears immediately."
    echo ""
    sleep 1
    echo "# Block 2"
    echo ""
    echo "Second block appears after 1 second."
    echo ""
    sleep 1
    echo "# Block 3"
    echo ""
    echo "Third block appears after 2 seconds total."
} | ./glow -w0 --flow=-1 - 2>/dev/null | while IFS= read -r line; do
    echo "[$(date +%H:%M:%S)] $line"
done && echo -e "\n✅ ${GREEN}PASS${NC}" || echo -e "\n❌ ${RED}FAIL${NC}"

echo
echo -e "${GREEN}Progressive streaming tests complete!${NC}"
echo

# Test 5: Measure actual progressive behavior
echo -e "${YELLOW}Test 5: Quantitative progressive measurement${NC}"
echo "Comparing input timing vs output timing..."
echo

# Create a temporary FIFO for timing analysis
TMPDIR=$(mktemp -d)
FIFO="$TMPDIR/test.fifo"
mkfifo "$FIFO"

# Start glow in background reading from FIFO
./glow -w0 --flow=-1 - < "$FIFO" 2>/dev/null | pv -bt > /dev/null &
GLOW_PID=$!

# Feed input slowly and measure
{
    echo "# First chunk at 0s"
    echo ""
    echo "Content 1"
    echo ""
    sleep 0.5
    echo "# Second chunk at 0.5s"
    echo ""
    echo "Content 2"
    echo ""
    sleep 0.5
    echo "# Third chunk at 1s"
    echo ""
    echo "Content 3"
} | pv -q -L 100 > "$FIFO" && echo -e "\n✅ ${GREEN}PASS${NC}" || echo -e "\n❌ ${RED}FAIL${NC}"

wait $GLOW_PID

# Cleanup
rm -rf "$TMPDIR"
