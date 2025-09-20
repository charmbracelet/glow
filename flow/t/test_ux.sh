#!/bin/bash

set -uo pipefail

i=0 x=-1 a=-1 y=-1 n=0 all=-1 yes=-1 no=0 on=-1 off=0
[[ ${V-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || V=y
[[ ${X-} =~ ^(-?[0-9]+|[ixayn]|all|yes|no|on|off)$ ]] || X=n

it () { echo -e "$(printf '%.0s=' {1..79})\nUX TEST $((++i)) $*\n$(printf '%.0s-' {1..79})"; } 1>&2
ok () { echo UX TEST $i $((( $? )) && echo ❌ || echo ✅; echo " V=$V X=$X"); } 1>&2
vv () { ((i==V || V<0)) && cat -u || cat -u > /dev/null; }

# Build fresh
go build .
cp -avn ./glow ./glow.orig >&2 || true
timeout -s KILL 3s ./glow -w0 - < /dev/null > /dev/null || exit 3


# ==================== REAL-WORLD USAGE SCENARIOS ====================

it ux_test_1: should handle log tailing simulation gracefully; (
  ((i==X || X<0)) && set -x; set -e

  # Simulate log file with mixed content arriving over time
  {
    echo "# System Log"
    echo ""
    sleep 0.1
    echo "## $(date '+%Y-%m-%d %H:%M:%S') - Service Started"
    sleep 0.1
    echo ""
    echo "- Database connection established"
    echo "- Cache warming complete"
    sleep 0.1
    echo ""
    echo "\`\`\`json"
    echo '{"status": "healthy", "uptime": 30}'
    echo "\`\`\`"
    sleep 0.1
    echo ""
    echo "## Metrics"
    echo "- Memory: 45MB"
    echo "- CPU: 12%"
  } | timeout 2s ./glow -w0 --flow=1024 - | head -20 > /dev/null

  # Should complete without hanging or error
) | vv; ok

it ux_test_2: should feel immediately responsive on first input; (
  ((i==X || X<0)) && set -x; set -e

  # Test perception of immediate output
  start_time=$(date +%s%N)
  echo "# Hello World" | ./glow -w0 --flow=0 - > /tmp/ux_output &
  glow_pid=$!

  # Wait until we see output
  while [[ ! -s /tmp/ux_output ]] && kill -0 $glow_pid 2>/dev/null; do
    sleep 0.001
  done
  end_time=$(date +%s%N)

  wait $glow_pid

  # Should respond within 100ms (100,000,000 nanoseconds)
  response_time=$((end_time - start_time))
  [[ $response_time -lt 100000000 ]]

) | vv; ok

it ux_test_3: should handle git log style input naturally; (
  ((i==X || X<0)) && set -x; set -e

  # Simulate git log --oneline | glow pattern
  {
    echo "# Recent Commits"
    echo ""
    for i in {1..5}; do
      echo "## $(printf '%07x' $RANDOM) feat: implement feature $i"
      echo ""
      echo "- Added new functionality"
      echo "- Updated tests"
      echo "- Documentation changes"
      echo ""
      sleep 0.05
    done
  } | timeout 1s ./glow -w0 --flow=2048 - > /dev/null

  # Should complete gracefully
) | vv; ok

it ux_test_4: should handle live documentation updates smoothly; (
  ((i==X || X<0)) && set -x; set -e

  # Simulate live doc updates
  {
    echo "# Live Documentation"
    echo ""
    echo "Last updated: $(date)"
    echo ""
    for section in "Setup" "Configuration" "Usage" "Troubleshooting"; do
      echo "## $section"
      echo ""
      echo "Content for $section section..."
      echo ""
      echo "\`\`\`bash"
      echo "# Example command for $section"
      echo "echo 'Working on $section'"
      echo "\`\`\`"
      echo ""
      sleep 0.1
    done
  } | timeout 3s ./glow -w0 --flow=0 - > /dev/null

) | vv; ok

# ==================== PERFORMANCE PERCEPTION TESTS ====================

it ux_test_5: should maintain streaming feel with large single block; (
  ((i==X || X<0)) && set -x; set -e

  # Create large list that should stream smoothly
  {
    echo "# Large Task List"
    echo ""
    for i in {1..1000}; do
      echo "- [ ] Task $i: Lorem ipsum dolor sit amet"
      # Occasional flush to test buffering
      ((i % 100 == 0)) && sleep 0.01
    done
  } | timeout 5s ./glow -w0 --flow=4096 - | head -50 > /dev/null

  # Should not hang or consume excessive memory
) | vv; ok

it ux_test_6: should handle rapid small updates gracefully; (
  ((i==X || X<0)) && set -x; set -e

  # Simulate rapid fire updates like a chat log
  {
    echo "# Message Log"
    echo ""
    for i in {1..50}; do
      echo "**User$((i % 5))**: Message $i at $(date +%H:%M:%S)"
      echo ""
      sleep 0.01
    done
  } | timeout 3s ./glow -w0 --flow=512 - > /dev/null

) | vv; ok

# ==================== MEMORY PRESSURE FROM UX PERSPECTIVE ====================

it ux_test_7: should handle massive input without user-visible slowdown; (
  ((i==X || X<0)) && set -x; set -e

  # Generate large document that should stay memory-bounded
  {
    echo "# Performance Test Document"
    echo ""
    for section in {1..100}; do
      echo "## Section $section"
      echo ""
      echo "This is section $section with some content."
      echo ""
      echo "\`\`\`text"
      for line in {1..10}; do
        echo "Line $line of section $section: $(printf '%.0sX' {1..100})"
      done
      echo "\`\`\`"
      echo ""
    done
  } | timeout 10s ./glow -w0 --flow=8192 - | head -100 > /dev/null

  # Should complete without memory issues
) | vv; ok

it ux_test_8: should handle zero buffer gracefully under pressure; (
  ((i==X || X<0)) && set -x; set -e

  # Test immediate flushing with substantial content
  {
    echo "# Zero Buffer Test"
    for i in {1..20}; do
      echo ""
      echo "## Header $i"
      echo ""
      echo "Paragraph with content for section $i."
      echo ""
      echo "\`\`\`"
      echo "code block $i"
      echo "\`\`\`"
      sleep 0.02
    done
  } | timeout 5s ./glow -w0 --flow=0 - > /dev/null

) | vv; ok

# ==================== EDGE CASES USERS ENCOUNTER ====================

it ux_test_9: should handle mixed content gracefully; (
  ((i==X || X<0)) && set -x; set -e

  # Real-world mixed content that often breaks parsers
  {
    echo "# Mixed Content Test"
    echo ""
    echo "Normal paragraph with **bold** and *italic*."
    echo ""
    echo "HTML that might be ignored: <img src='test.jpg' alt='test'>"
    echo ""
    echo "| Table | Header |"
    echo "|-------|--------|"
    echo "| Cell1 | Cell2  |"
    echo ""
    echo "\`\`\`javascript"
    echo "// Code with special characters"
    echo "const test = 'string with \\\"escapes\\\"';"
    echo "\`\`\`"
    echo ""
    echo "> Blockquote with **formatting**"
    echo ""
    echo "1. Numbered list"
    echo "   - Nested item"
    echo "   - Another nested"
    echo "2. Second number"
  } | ./glow -w0 --flow=1024 - > /dev/null

) | vv; ok

it ux_test_10: should handle empty and near-empty input gracefully; (
  ((i==X || X<0)) && set -x; set -e

  # Edge cases that users might encounter

  # Completely empty
  echo -n "" | ./glow -w0 --flow=0 - > /dev/null

  # Just whitespace
  echo -e "   \n  \n   " | ./glow -w0 --flow=0 - > /dev/null

  # Single character with no newline
  echo -n "a" | ./glow -w0 --flow=0 - > /dev/null

  # Single character
  echo "a" | ./glow -w0 --flow=0 - > /dev/null

  # Just HTML (produces no visible output)
  echo "<div>invisible</div>" | ./glow -w0 --flow=0 - > /dev/null

) | vv; ok

# ==================== SIGNAL HANDLING FROM UX PERSPECTIVE ====================

it ux_test_11: should preserve user work on interruption; (
  ((i==X || X<0)) && set -x; set -e

  # Simulate user interrupting mid-stream
  {
    echo "# Important Document"
    echo ""
    echo "Critical content that user doesn't want to lose..."
    echo ""
    sleep 0.2
    echo "More important content..."
    sleep 0.5
    echo "This might not be seen..."
  } | timeout -s INT 0.3s ./glow -w0 --flow=1024 - > /tmp/interrupted_output 2>/dev/null || true

  # Should have captured the initial content
  grep -q "Important Document" /tmp/interrupted_output
  grep -q "Critical content" /tmp/interrupted_output

) | vv; ok

it ux_test_12: should handle pipe broken gracefully; (
  ((i==X || X<0)) && set -x; set -e

  # Simulate downstream pipe closure (like less quitting)
  {
    for i in {1..100}; do
      echo "# Section $i"
      echo "Content for section $i"
      echo ""
    done
  } | ./glow -w0 --flow=1024 - | head -5 > /dev/null

  # Should not error when pipe breaks
) | vv; ok

echo "UX validation tests completed!"
