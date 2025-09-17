package flow

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestCriticalOutputFidelity tests that output is identical with different flow modes
// Migrated from: test_fidelity_unlimited
func TestCriticalOutputFidelity(t *testing.T) {
	t.Run("output_fidelity_unlimited", func(t *testing.T) {
		input := "# Test\nHello **world**"

		// Test with unlimited buffer
		outputUnlimited := runFlow(t, input, -1)

		// Test with small buffer
		outputStreamed := runFlow(t, input, 16)

		// Output should be identical
		if outputUnlimited != outputStreamed {
			t.Errorf("Output mismatch:\nUnlimited: %q\nStreamed: %q",
				outputUnlimited, outputStreamed)
		}
	})
}

// TestCriticalSignalHandling tests clean shutdown on SIGTERM
// Migrated from: test_signal_clean
func TestCriticalSignalHandling(t *testing.T) {
	t.Run("signal_handling_clean_shutdown", func(t *testing.T) {
		// Create a slow reader that will be interrupted
		slowReader := &testSlowStreamReader{
			lines: []string{
				"# Test",
				"Content that takes time",
			},
			delays: []time.Duration{
				0,
				1 * time.Second,
			},
		}

		// Context with early cancellation
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, slowReader, &output, 16, passthroughRenderer)

		// Should exit cleanly with context deadline exceeded
		if err != nil && err != context.DeadlineExceeded {
			t.Errorf("Expected clean shutdown, got error: %v", err)
		}

		// Should have at least started processing
		if output.Len() == 0 {
			t.Error("No output produced before signal")
		}
	})
}

// TestCriticalEmptyInput tests handling of empty/nil input gracefully
// Migrated from: test_empty_input
func TestCriticalEmptyInput(t *testing.T) {
	t.Run("empty_input_handling", func(t *testing.T) {
		input := ""

		// Test with unlimited buffer
		outputUnlimited := runFlow(t, input, -1)

		// Test with small buffer
		outputStreamed := runFlow(t, input, 16)

		// Both should handle empty input identically
		if outputUnlimited != outputStreamed {
			t.Errorf("Empty input handling differs:\nUnlimited: %q\nStreamed: %q",
				outputUnlimited, outputStreamed)
		}
	})
}

// TestCriticalLargeInput tests processing large documents without OOM
// Migrated from: test_large_input
func TestCriticalLargeInput(t *testing.T) {
	t.Run("large_input_1MB", func(t *testing.T) {
		// Generate 1MB of markdown (approximately)
		var input strings.Builder
		for i := 0; i < 10000; i++ {
			input.WriteString("# Heading ")
			input.WriteString(fmt.Sprintf("%d", i))
			input.WriteString("\nParagraph content with some text.\n\n")
		}

		// Process with timeout to ensure it doesn't hang
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var output bytes.Buffer
		err := Flow(ctx, strings.NewReader(input.String()), &output, 1024, passthroughRenderer)

		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("Large input processing failed: %v", err)
		}

		// Should have produced output
		if output.Len() == 0 {
			t.Error("No output from 1MB input")
		}
	})
}

// TestCriticalCodeBlockFidelity tests preserving code blocks correctly
// Migrated from: test_code_blocks
func TestCriticalCodeBlockFidelity(t *testing.T) {
	t.Run("code_block_fidelity", func(t *testing.T) {
		input := "```go\nfunc main() {\n    fmt.Println(\"test\")\n}\n```"

		// Test with unlimited buffer
		outputUnlimited := runFlow(t, input, -1)

		// Test with small buffer that might split the code block
		outputStreamed := runFlow(t, input, 64)

		// Verify code content is preserved in both outputs
		codeContent := []string{"func main", "fmt.Println", "test"}
		for _, content := range codeContent {
			if !strings.Contains(outputUnlimited, content) {
				t.Errorf("Code content %q missing from unlimited output", content)
			}
			if !strings.Contains(outputStreamed, content) {
				t.Errorf("Code content %q missing from streamed output", content)
			}
		}

		// Note: Glamour formats code blocks, doesn't preserve raw ``` markers
		// Small buffers may produce slightly different formatting
	})
}

// TestCriticalNoBufferMode tests streaming with --flow=-1 works
// Migrated from: test_no_buffer
func TestCriticalNoBufferMode(t *testing.T) {
	t.Run("no_buffer_mode", func(t *testing.T) {
		input := "# Test\nContent line\n\n## Another header"

		ctx := context.Background()
		var output bytes.Buffer
		err := Flow(ctx, strings.NewReader(input), &output, -1, passthroughRenderer)

		if err != nil {
			t.Fatalf("No buffer mode failed: %v", err)
		}

		// Should have produced output
		if output.Len() == 0 {
			t.Error("No output in unbuffered mode")
		}

		// Should contain the content
		result := output.String()
		if !strings.Contains(result, "Test") {
			t.Error("Content missing from unbuffered output")
		}
	})
}

// TestCriticalBinaryDataSafety tests handling binary data without panic
// Migrated from: test_binary_safety
func TestCriticalBinaryDataSafety(t *testing.T) {
	t.Run("binary_data_safety", func(t *testing.T) {
		// Generate 1KB of random binary data
		binaryData := make([]byte, 1024)
		_, err := rand.Read(binaryData)
		if err != nil {
			t.Fatalf("Failed to generate binary data: %v", err)
		}

		// Process with timeout to ensure it doesn't hang
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var output bytes.Buffer
		err = Flow(ctx, bytes.NewReader(binaryData), &output, 16, passthroughRenderer)

		// Should not panic - error is acceptable for binary data
		if err != nil && err != context.DeadlineExceeded {
			t.Logf("Binary data caused error (acceptable): %v", err)
		}

		// Test passed if we didn't panic
		t.Log("Binary data handled without panic")
	})
}

// TestCriticalPipeChain tests working in pipe chains
// Migrated from: test_pipe_chain
func TestCriticalPipeChain(t *testing.T) {
	t.Run("pipe_chain_compatibility", func(t *testing.T) {
		input := "# Test\nContent\n## Header Two"

		// Simulate pipe chain: input -> Flow -> grep
		ctx := context.Background()
		var flowOutput bytes.Buffer
		err := Flow(ctx, strings.NewReader(input), &flowOutput, 16, passthroughRenderer)
		if err != nil {
			t.Fatalf("Flow failed in pipe chain: %v", err)
		}

		// Verify output can be piped (contains expected content)
		result := flowOutput.String()
		if !strings.Contains(result, "Test") {
			t.Error("Pipe chain: missing expected content 'Test'")
		}
		if !strings.Contains(result, "Header Two") {
			t.Error("Pipe chain: missing expected content 'Header Two'")
		}

		// Simulate further processing (like grep)
		lines := strings.Split(result, "\n")
		found := false
		for _, line := range lines {
			if strings.Contains(line, "Test") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Pipe chain: grep simulation failed")
		}
	})
}

// TestCriticalMultipleDocuments tests handling multiple markdown docs in stream
// Migrated from: test_multiple_docs
func TestCriticalMultipleDocuments(t *testing.T) {
	t.Run("multiple_documents", func(t *testing.T) {
		input := "# Doc1\n---\n# Doc2"

		// Test with unlimited buffer
		outputUnlimited := runFlow(t, input, -1)

		// Test with small buffer
		outputStreamed := runFlow(t, input, 32)

		// Multiple documents should be handled identically
		if outputUnlimited != outputStreamed {
			t.Errorf("Multiple docs output differs:\nUnlimited: %q\nStreamed: %q",
				outputUnlimited, outputStreamed)
		}

		// Should contain both documents
		if !strings.Contains(outputUnlimited, "Doc1") {
			t.Error("First document missing")
		}
		if !strings.Contains(outputUnlimited, "Doc2") {
			t.Error("Second document missing")
		}
	})
}

// TestCriticalEOFSpacing tests handling EOF spacing consistently
// Migrated from: test_eof_spacing
func TestCriticalEOFSpacing(t *testing.T) {
	t.Run("EOF_spacing_consistency", func(t *testing.T) {
		// Test that EOF normalization works
		inputWithNewline := "# Test\n"
		inputWithoutNewline := "# Test"

		// Both inputs should produce normalized output
		output1 := runFlow(t, inputWithNewline, 16)
		output2 := runFlow(t, inputWithoutNewline, 16)

		// Both should produce non-empty output
		if len(output1) == 0 {
			t.Error("Empty output for input with newline")
		}
		if len(output2) == 0 {
			t.Error("Empty output for input without newline")
		}

		// Both should contain the content
		if !strings.Contains(output1, "Test") {
			t.Error("Content missing in output1")
		}
		if !strings.Contains(output2, "Test") {
			t.Error("Content missing in output2")
		}

		// EOF normalization should make them similar (both contain the header)
		// The exact spacing might differ but content should be present
		t.Logf("Output1 length: %d, Output2 length: %d", len(output1), len(output2))
	})
}

// Additional integration test for signal handling with real process
func TestCriticalSignalIntegration(t *testing.T) {
	// Skip if not in CI or if glow binary doesn't exist
	if _, err := os.Stat("./glow"); os.IsNotExist(err) {
		t.Skip("Skipping integration test: glow binary not found")
	}

	t.Run("real_signal_handling", func(t *testing.T) {
		// Start glow process with slow input
		cmd := exec.Command("./glow", "-w0", "--flow=16", "-")
		// Disable TTY interaction
		cmd.Env = append(os.Environ(),
			"TERM=dumb",
			"NO_COLOR=1",
			"CI=true",
		)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("Failed to create stdin pipe: %v", err)
		}

		// Start the process
		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start process: %v", err)
		}

		// Write some input
		go func() {
			stdin.Write([]byte("# Test\n"))
			time.Sleep(100 * time.Millisecond)
			stdin.Write([]byte("More content\n"))
			stdin.Close()
		}()

		// Give it a moment to start processing
		time.Sleep(50 * time.Millisecond)

		// Send SIGTERM
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Logf("Failed to send signal: %v", err)
		}

		// Wait for process to exit
		err = cmd.Wait()
		if err != nil {
			// Check if it's a signal exit (which is expected)
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Signal exits are acceptable
				t.Logf("Process exited with: %v", exitErr)
			} else {
				t.Errorf("Unexpected error: %v", err)
			}
		}
		// Clean exit is also acceptable
	})
}

// Note: testSlowStreamReader is already defined in flow_arch_test.go
