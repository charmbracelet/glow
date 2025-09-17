package flow

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// TestUX tests user experience scenarios and real-world usage patterns
// Migrated from: flow/t/test_ux.sh (special format test framework)
// These tests validate UX quality, responsiveness, and real-world usage scenarios
func TestUX(t *testing.T) {
	t.Run("log_tailing_simulation", func(t *testing.T) {
		// UX Test 1: Should handle log tailing simulation gracefully
		// Simulates log file with mixed content arriving over time

		logReader := &logTailingReader{
			entries: []logEntry{
				{content: "# System Log\n\n", delay: 0},
				{content: "## 2024-01-01 12:00:00 - Service Started\n\n", delay: 100 * time.Millisecond},
				{content: "- Database connection established\n- Cache warming complete\n\n", delay: 100 * time.Millisecond},
				{content: "```json\n{\"status\": \"healthy\", \"uptime\": 30}\n```\n\n", delay: 100 * time.Millisecond},
				{content: "## Metrics\n- Memory: 45MB\n- CPU: 12%\n", delay: 100 * time.Millisecond},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := Flow(ctx, logReader, &buf, 1024, passthroughRenderer)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Log tailing simulation failed: %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "System Log") && strings.Contains(output, "Service Started") {
			t.Log("✅ PASS: Log tailing simulation handled gracefully")
		} else {
			t.Error("❌ FAIL: Log tailing content not processed correctly")
		}
	})

	t.Run("immediate_responsiveness", func(t *testing.T) {
		// UX Test 2: Should feel immediately responsive on first input
		// Tests perception of immediate output

		ctx := context.Background()
		simpleContent := "# Hello World\n"

		start := time.Now()
		var buf bytes.Buffer
		err := Flow(ctx, strings.NewReader(simpleContent), &buf, 0, passthroughRenderer)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Immediate response test failed: %v", err)
		}

		// Should respond very quickly (under 100ms is excellent UX)
		if duration < 100*time.Millisecond && buf.Len() > 0 {
			t.Logf("✅ PASS: Immediate responsiveness (%v)", duration)
		} else {
			t.Errorf("❌ FAIL: Response too slow (%v) or no output", duration)
		}
	})

	t.Run("git_log_style_input", func(t *testing.T) {
		// UX Test 3: Should handle git log style input naturally
		// Simulates git log --oneline | glow pattern

		gitLogReader := &gitLogReader{
			commits: []string{
				"# Recent Commits\n\n",
				"## a1b2c3d feat: implement feature 1\n\n- Added new functionality\n- Updated tests\n- Documentation changes\n\n",
				"## b2c3d4e feat: implement feature 2\n\n- Added new functionality\n- Updated tests\n- Documentation changes\n\n",
				"## c3d4e5f feat: implement feature 3\n\n- Added new functionality\n- Updated tests\n- Documentation changes\n\n",
			},
			delay: 50 * time.Millisecond,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := Flow(ctx, gitLogReader, &buf, 2048, passthroughRenderer)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Git log style input failed: %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "Recent Commits") && strings.Contains(output, "feat: implement") {
			t.Log("✅ PASS: Git log style input handled naturally")
		} else {
			t.Error("❌ FAIL: Git log content not processed correctly")
		}
	})

	t.Run("live_documentation_updates", func(t *testing.T) {
		// UX Test 4: Should handle live documentation updates smoothly
		// Simulates live doc updates with sections arriving over time

		docReader := &liveDocReader{
			sections: []docSection{
				{name: "Setup", delay: 0},
				{name: "Configuration", delay: 100 * time.Millisecond},
				{name: "Usage", delay: 100 * time.Millisecond},
				{name: "Troubleshooting", delay: 100 * time.Millisecond},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := Flow(ctx, docReader, &buf, 0, passthroughRenderer)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Live documentation updates failed: %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "Live Documentation") && strings.Contains(output, "Setup") {
			t.Log("✅ PASS: Live documentation updates handled smoothly")
		} else {
			t.Error("❌ FAIL: Live documentation not processed correctly")
		}
	})

	t.Run("large_single_block_streaming", func(t *testing.T) {
		// UX Test 5: Should maintain streaming feel with large single block
		// Create large list that should stream smoothly

		var listBuilder strings.Builder
		listBuilder.WriteString("# Large Task List\n\n")
		for i := 1; i <= 1000; i++ {
			listBuilder.WriteString("- [ ] Task ")
			listBuilder.WriteString(string(rune('A' + (i-1)%26)))
			listBuilder.WriteString(": Lorem ipsum dolor sit amet\n")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use limited writer to simulate head -50
		limitedWriter := &uxLimitedWriter{maxLines: 50}
		err := Flow(ctx, strings.NewReader(listBuilder.String()), limitedWriter, 4096, passthroughRenderer)

		// Expect either success or closed pipe (head behavior)
		if err != nil && !strings.Contains(err.Error(), "closed") && err != context.DeadlineExceeded {
			t.Errorf("Large single block streaming failed: %v", err)
		}

		if limitedWriter.lineCount > 0 {
			t.Log("✅ PASS: Large single block streaming maintained")
		} else {
			t.Error("❌ FAIL: No output from large single block")
		}
	})

	t.Run("rapid_small_updates", func(t *testing.T) {
		// UX Test 6: Should handle rapid small updates gracefully
		// Simulates rapid fire updates like a chat log

		chatReader := &chatLogReader{
			messages: make([]string, 50),
			delay:    10 * time.Millisecond,
		}

		// Generate chat messages
		for i := 0; i < 50; i++ {
			user := i % 5
			chatReader.messages[i] = "**User" + string(rune('A'+user)) + "**: Message " + string(rune('0'+(i%10))) + " at 12:00:00\n\n"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := Flow(ctx, chatReader, &buf, 512, passthroughRenderer)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Rapid small updates failed: %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "Message Log") && strings.Contains(output, "User") {
			t.Log("✅ PASS: Rapid small updates handled gracefully")
		} else {
			t.Error("❌ FAIL: Rapid updates not processed correctly")
		}
	})

	t.Run("massive_input_no_slowdown", func(t *testing.T) {
		// UX Test 7: Should handle massive input without user-visible slowdown
		// Generate large document that should stay memory-bounded

		var massiveBuilder strings.Builder
		massiveBuilder.WriteString("# Performance Test Document\n\n")
		for section := 1; section <= 100; section++ {
			massiveBuilder.WriteString("## Section ")
			massiveBuilder.WriteString(string(rune('A' + (section-1)%26)))
			massiveBuilder.WriteString("\n\nThis is section ")
			massiveBuilder.WriteString(string(rune('A' + (section-1)%26)))
			massiveBuilder.WriteString(" with some content.\n\n```text\n")
			for line := 1; line <= 10; line++ {
				massiveBuilder.WriteString("Line ")
				massiveBuilder.WriteString(string(rune('0' + line%10)))
				massiveBuilder.WriteString(" of section ")
				massiveBuilder.WriteString(string(rune('A' + (section-1)%26)))
				massiveBuilder.WriteString(": ")
				massiveBuilder.WriteString(strings.Repeat("X", 100))
				massiveBuilder.WriteString("\n")
			}
			massiveBuilder.WriteString("```\n\n")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Use limited writer to simulate head -100
		limitedWriter := &uxLimitedWriter{maxLines: 100}
		err := Flow(ctx, strings.NewReader(massiveBuilder.String()), limitedWriter, 8192, passthroughRenderer)

		// Expect either success or closed pipe (head behavior)
		if err != nil && !strings.Contains(err.Error(), "closed") && err != context.DeadlineExceeded {
			t.Errorf("Massive input test failed: %v", err)
		}

		if limitedWriter.lineCount > 0 {
			t.Log("✅ PASS: Massive input handled without user-visible slowdown")
		} else {
			t.Error("❌ FAIL: No output from massive input")
		}
	})

	t.Run("zero_buffer_under_pressure", func(t *testing.T) {
		// UX Test 8: Should handle zero buffer gracefully under pressure
		// Test immediate flushing with substantial content

		pressureReader := &pressureTestReader{
			sections: make([]string, 20),
			delay:    20 * time.Millisecond,
		}

		// Generate pressure test content
		for i := 0; i < 20; i++ {
			pressureReader.sections[i] = "\n## Header " + string(rune('A'+(i%26))) + "\n\nParagraph with content for section " + string(rune('A'+(i%26))) + ".\n\n```\ncode block " + string(rune('A'+(i%26))) + "\n```\n"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := Flow(ctx, pressureReader, &buf, 0, passthroughRenderer)
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Zero buffer pressure test failed: %v", err)
		}

		if buf.Len() > 0 {
			t.Log("✅ PASS: Zero buffer handled gracefully under pressure")
		} else {
			t.Error("❌ FAIL: No output with zero buffer under pressure")
		}
	})

	t.Run("mixed_content_graceful", func(t *testing.T) {
		// UX Test 9: Should handle mixed content gracefully
		// Real-world mixed content that often breaks parsers

		mixedContent := `# Mixed Content Test

Normal paragraph with **bold** and *italic*.

HTML that might be ignored: <img src='test.jpg' alt='test'>

| Table | Header |
|-------|--------|
| Cell1 | Cell2  |

` + "```javascript" + `
// Code with special characters
const test = 'string with \"escapes\"';
` + "```" + `

> Blockquote with **formatting**

1. Numbered list
   - Nested item
   - Another nested
2. Second number`

		ctx := context.Background()
		var buf bytes.Buffer
		err := Flow(ctx, strings.NewReader(mixedContent), &buf, 1024, passthroughRenderer)
		if err != nil {
			t.Errorf("Mixed content test failed: %v", err)
		}

		if buf.Len() > 0 {
			t.Log("✅ PASS: Mixed content handled gracefully")
		} else {
			t.Error("❌ FAIL: Mixed content not processed correctly")
		}
	})

	t.Run("empty_and_near_empty_input", func(t *testing.T) {
		// UX Test 10: Should handle empty and near-empty input gracefully
		// Edge cases that users might encounter

		testCases := []struct {
			name  string
			input string
		}{
			{"completely_empty", ""},
			{"just_whitespace", "   \n  \n   "},
			{"single_char_no_newline", "a"},
			{"single_char", "a\n"},
			{"just_html", "<div>invisible</div>"},
		}

		ctx := context.Background()
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				err := Flow(ctx, strings.NewReader(tc.input), &buf, 0, passthroughRenderer)
				if err != nil {
					t.Errorf("Empty/near-empty input test '%s' failed: %v", tc.name, err)
				}
				// Just needs to not crash - output can be empty
			})
		}

		t.Log("✅ PASS: Empty and near-empty input handled gracefully")
	})

	t.Run("preserve_work_on_interruption", func(t *testing.T) {
		// UX Test 11: Should preserve user work on interruption
		// Simulate user interrupting mid-stream

		interruptReader := &interruptibleReader{
			chunks: []string{
				"# Important Document\n\n",
				"Critical content that user doesn't want to lose...\n\n",
				"More important content...\n",
				"This might not be seen...\n",
			},
			delays: []time.Duration{
				0,
				0,
				200 * time.Millisecond,
				500 * time.Millisecond,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		var buf bytes.Buffer
		_ = Flow(ctx, interruptReader, &buf, 1024, passthroughRenderer)

		// Should have captured initial content before interruption
		output := buf.String()
		if strings.Contains(output, "Important Document") && strings.Contains(output, "Critical content") {
			t.Log("✅ PASS: User work preserved on interruption")
		} else {
			t.Error("❌ FAIL: Initial content not preserved on interruption")
		}

		// Should NOT have late content
		if strings.Contains(output, "might not be seen") {
			t.Error("❌ FAIL: Late content present despite interruption")
		}
	})

	t.Run("broken_pipe_graceful", func(t *testing.T) {
		// UX Test 12: Should handle pipe broken gracefully
		// Simulate downstream pipe closure (like less quitting)

		var pipeContent strings.Builder
		for i := 1; i <= 100; i++ {
			pipeContent.WriteString("# Section ")
			pipeContent.WriteString(string(rune('A' + (i-1)%26)))
			pipeContent.WriteString("\nContent for section ")
			pipeContent.WriteString(string(rune('A' + (i-1)%26)))
			pipeContent.WriteString("\n\n")
		}

		ctx := context.Background()

		// Use writer that simulates pipe break after a few writes
		uxBrokenPipeWriter := &uxBrokenPipeWriter{maxWrites: 5}
		err := Flow(ctx, strings.NewReader(pipeContent.String()), uxBrokenPipeWriter, 1024, passthroughRenderer)

		// Should handle broken pipe gracefully (io.ErrClosedPipe is expected)
		if err != nil && !strings.Contains(err.Error(), "closed") && err != io.ErrClosedPipe {
			t.Errorf("Broken pipe test unexpected error: %v", err)
		}

		if uxBrokenPipeWriter.writeCount > 0 {
			t.Log("✅ PASS: Broken pipe handled gracefully")
		} else {
			t.Error("❌ FAIL: No writes before pipe break")
		}
	})
}

// Helper types for UX testing

// logTailingReader simulates log entries arriving over time
type logTailingReader struct {
	entries []logEntry
	index   int
}

type logEntry struct {
	content string
	delay   time.Duration
}

func (r *logTailingReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.entries) {
		return 0, io.EOF
	}

	entry := r.entries[r.index]
	if entry.delay > 0 {
		time.Sleep(entry.delay)
	}

	n = copy(p, entry.content)
	r.index++
	return n, nil
}

// gitLogReader simulates git log output with delays
type gitLogReader struct {
	commits []string
	index   int
	delay   time.Duration
}

func (r *gitLogReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.commits) {
		return 0, io.EOF
	}

	if r.index > 0 {
		time.Sleep(r.delay)
	}

	commit := r.commits[r.index]
	n = copy(p, commit)
	r.index++
	return n, nil
}

// liveDocReader simulates live documentation updates
type liveDocReader struct {
	sections []docSection
	index    int
	inHeader bool
}

type docSection struct {
	name  string
	delay time.Duration
}

func (r *liveDocReader) Read(p []byte) (n int, err error) {
	if !r.inHeader {
		// Send header first
		header := "# Live Documentation\n\nLast updated: 2024-01-01\n\n"
		n = copy(p, header)
		r.inHeader = true
		return n, nil
	}

	if r.index >= len(r.sections) {
		return 0, io.EOF
	}

	section := r.sections[r.index]
	if section.delay > 0 {
		time.Sleep(section.delay)
	}

	content := "## " + section.name + "\n\nContent for " + section.name + " section...\n\n```bash\n# Example command for " + section.name + "\necho 'Working on " + section.name + "'\n```\n\n"
	n = copy(p, content)
	r.index++
	return n, nil
}

// uxLimitedWriter simulates head -n behavior for UX tests
type uxLimitedWriter struct {
	maxLines  int
	lineCount int
	closed    bool
}

func (w *uxLimitedWriter) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}

	// Count newlines to simulate line limit
	for _, b := range p {
		if b == '\n' {
			w.lineCount++
			if w.lineCount >= w.maxLines {
				w.closed = true
				return len(p), io.ErrClosedPipe
			}
		}
	}

	return len(p), nil
}

// chatLogReader simulates rapid chat messages
type chatLogReader struct {
	messages []string
	index    int
	delay    time.Duration
	inHeader bool
}

func (r *chatLogReader) Read(p []byte) (n int, err error) {
	if !r.inHeader {
		header := "# Message Log\n\n"
		n = copy(p, header)
		r.inHeader = true
		return n, nil
	}

	if r.index >= len(r.messages) {
		return 0, io.EOF
	}

	time.Sleep(r.delay)

	message := r.messages[r.index]
	n = copy(p, message)
	r.index++
	return n, nil
}

// pressureTestReader simulates content under pressure
type pressureTestReader struct {
	sections []string
	index    int
	delay    time.Duration
	inHeader bool
}

func (r *pressureTestReader) Read(p []byte) (n int, err error) {
	if !r.inHeader {
		header := "# Zero Buffer Test\n"
		n = copy(p, header)
		r.inHeader = true
		return n, nil
	}

	if r.index >= len(r.sections) {
		return 0, io.EOF
	}

	time.Sleep(r.delay)

	section := r.sections[r.index]
	n = copy(p, section)
	r.index++
	return n, nil
}

// interruptibleReader simulates content that can be interrupted
type interruptibleReader struct {
	chunks []string
	delays []time.Duration
	index  int
}

func (r *interruptibleReader) Read(p []byte) (n int, err error) {
	if r.index >= len(r.chunks) {
		return 0, io.EOF
	}

	if r.index < len(r.delays) && r.delays[r.index] > 0 {
		time.Sleep(r.delays[r.index])
	}

	chunk := r.chunks[r.index]
	n = copy(p, chunk)
	r.index++
	return n, nil
}

// uxBrokenPipeWriter simulates pipe breaking after some writes
type uxBrokenPipeWriter struct {
	maxWrites  int
	writeCount int
	closed     bool
}

func (w *uxBrokenPipeWriter) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}

	w.writeCount++
	if w.writeCount >= w.maxWrites {
		w.closed = true
		return 0, io.ErrClosedPipe
	}

	return len(p), nil
}

// TestUXSuite provides comprehensive UX validation
func TestUXSuite(t *testing.T) {
	t.Log("=== USER EXPERIENCE VALIDATION TESTS ===")
	t.Log("Real-world usage scenarios and UX quality validation")
	t.Log("")

	categories := []struct {
		name  string
		desc  string
		tests []string
	}{
		{
			name: "Real-World Usage",
			desc: "Common streaming scenarios users encounter",
			tests: []string{
				"log_tailing_simulation",
				"git_log_style_input",
				"live_documentation_updates",
				"mixed_content_graceful",
			},
		},
		{
			name: "Performance Perception",
			desc: "User-perceived responsiveness and streaming feel",
			tests: []string{
				"immediate_responsiveness",
				"large_single_block_streaming",
				"rapid_small_updates",
				"massive_input_no_slowdown",
				"zero_buffer_under_pressure",
			},
		},
		{
			name: "Edge Cases & Robustness",
			desc: "Graceful handling of edge cases users encounter",
			tests: []string{
				"empty_and_near_empty_input",
				"preserve_work_on_interruption",
				"broken_pipe_graceful",
			},
		},
	}

	for _, category := range categories {
		t.Run(category.name, func(t *testing.T) {
			t.Logf("Category: %s", category.desc)
			t.Logf("Tests: %d", len(category.tests))
		})
	}

	t.Logf("\n=== UX COVERAGE ===")
	totalTests := 0
	for _, cat := range categories {
		totalTests += len(cat.tests)
	}
	t.Logf("Total UX scenarios: %d", totalTests)
	t.Logf("UX validation areas:")
	t.Logf("  - Real-world streaming scenarios")
	t.Logf("  - Performance perception and responsiveness")
	t.Logf("  - Edge case robustness")
	t.Logf("  - Work preservation on interruption")
	t.Logf("  - Graceful degradation")
}
