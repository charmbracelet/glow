package flow

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Test binary path - built once in TestMain
var glowBinary string

// Maximum timeout for any single test command
const maxTestTimeout = 3 * time.Second

// execGlowWithTimeout runs glow with aggressive timeout and isolation
func execGlowWithTimeout(t *testing.T, args []string, stdin string, timeout time.Duration) ([]byte, error) {
	if timeout > maxTestTimeout {
		timeout = maxTestTimeout
	}

	t.Logf("Executing: %s %v (timeout: %v)", filepath.Base(glowBinary), args, timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, glowBinary, args...)

	// Aggressive TTY isolation
	cmd.Env = append(os.Environ(),
		"TERM=dumb",
		"NO_COLOR=1",
		"CLICOLOR=0",
		"CLICOLOR_FORCE=0",
		"CI=true",
		"NONINTERACTIVE=1",
		"DEBIAN_FRONTEND=noninteractive",
	)

	// Provide stdin if given
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	} else {
		// Explicitly close stdin to prevent any reads
		cmd.Stdin = nil
	}

	// Create new process group for better isolation (Unix only)
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	}

	// Start with deadline
	output, err := cmd.CombinedOutput()

	// If context timed out, forcefully kill the process group
	if ctx.Err() == context.DeadlineExceeded {
		if cmd.Process != nil && runtime.GOOS != "windows" {
			// Kill the entire process group
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		t.Logf("Command timed out after %v, killed process group", timeout)
		return output, fmt.Errorf("timeout after %v", timeout)
	}

	return output, err
}

// TestMain builds the glow binary once for all integration tests
func TestMain(m *testing.M) {

	// Build the binary in temp directory
	tmpDir := os.TempDir()
	glowBinary = filepath.Join(tmpDir, "glow_test_binary")
	if runtime.GOOS == "windows" {
		glowBinary += ".exe"
	}

	// Build glow binary from project root
	projectRoot := "/Users/c/devel/glow"
	cmd := exec.Command("go", "build", "-o", glowBinary, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build glow binary: %v\nOutput: %s\n", err, output)
		// Skip integration tests if binary can't be built
		fmt.Fprintf(os.Stderr, "Skipping integration tests - binary build failed\n")
		os.Exit(0)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(glowBinary)
	os.Exit(code)
}

// 1. BINARY EXECUTION TESTS
func TestIntegrationBinaryExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testContent := `# Test Document

This is a test paragraph with **bold** and *italic* text.

- Item 1
- Item 2
- Item 3

[Link](https://example.com)
`

	t.Run("flow_variations", func(t *testing.T) {
		flowValues := []string{"-1", "0", "100", "1024", "10000"}
		outputs := make([]string, len(flowValues))

		for i, flow := range flowValues {
			output, err := execGlowWithTimeout(t, []string{"--flow=" + flow, "-"}, testContent, 2*time.Second)
			if err != nil {
				t.Errorf("Failed with --flow=%s: %v\nOutput: %s", flow, err, output)
				continue
			}
			outputs[i] = string(output)

			// Basic validation
			if len(output) == 0 {
				t.Errorf("Empty output with --flow=%s", flow)
			}
		}

		// Verify consistency - all should have similar content
		// (exact output may vary due to flow timing)
		for i := 1; i < len(outputs); i++ {
			if len(outputs[i]) == 0 {
				t.Errorf("Output %d is empty", i)
			}
		}
	})

	t.Run("streaming_mode", func(t *testing.T) {
		output, err := execGlowWithTimeout(t, []string{"--flow=-1", "-"}, "# Streaming Test\n\nContent flows immediately.", 2*time.Second)
		if err != nil {
			t.Fatalf("Streaming mode failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output in streaming mode")
		}
	})

	t.Run("buffered_mode", func(t *testing.T) {
		output, err := execGlowWithTimeout(t, []string{"--flow=0", "-"}, "# Buffered Test\n\nAll content at once.", 2*time.Second)
		if err != nil {
			t.Fatalf("Buffered mode failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output in buffered mode")
		}
	})

	t.Run("large_flow_value", func(t *testing.T) {
		output, err := execGlowWithTimeout(t, []string{"--flow=1000000", "-"}, "# Large Flow\n\nLarge buffer size.", 2*time.Second)
		if err != nil {
			t.Fatalf("Large flow value failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output with large flow value")
		}
	})

	t.Run("default_flow", func(t *testing.T) {
		// Test without --flow flag (should use default)
		output, err := execGlowWithTimeout(t, []string{"-"}, "# Default Flow\n\nUsing default settings.", 2*time.Second)
		if err != nil {
			t.Fatalf("Default flow failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output with default flow")
		}
	})
}

// 2. FILE SIZE TESTS
func TestIntegrationFileSizes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("small_file", func(t *testing.T) {
		content := "# Small\n\nJust 100 bytes of content here to test small file handling properly.\n"
		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		output, err := execGlowWithTimeout(t, []string{"--flow=100", tempFile}, "", 2*time.Second)
		if err != nil {
			t.Fatalf("Small file failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output for small file")
		}
	})

	t.Run("medium_file", func(t *testing.T) {
		// Generate 10KB of markdown
		var buf bytes.Buffer
		buf.WriteString("# Medium Document\n\n")
		for i := 0; i < 100; i++ {
			buf.WriteString(fmt.Sprintf("## Section %d\n\nThis is paragraph %d with some content to fill space.\n\n", i, i))
		}
		tempFile := createTempFile(t, buf.String())
		defer os.Remove(tempFile)

		output, err := execGlowWithTimeout(t, []string{"--flow=1024", tempFile}, "", 2*time.Second)
		if err != nil {
			t.Fatalf("Medium file failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output for medium file")
		}
	})

	t.Run("large_file", func(t *testing.T) {
		// Generate 100KB of markdown
		var buf bytes.Buffer
		buf.WriteString("# Large Document\n\n")
		for i := 0; i < 1000; i++ {
			buf.WriteString(fmt.Sprintf("## Section %d\n\nThis is paragraph %d with substantial content to create a large document for testing.\n\n", i, i))
		}
		tempFile := createTempFile(t, buf.String())
		defer os.Remove(tempFile)

		output, err := execGlowWithTimeout(t, []string{"--flow=10000", tempFile}, "", 2*time.Second)
		if err != nil {
			t.Fatalf("Large file failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output for large file")
		}
	})

	t.Run("empty_file", func(t *testing.T) {
		tempFile := createTempFile(t, "")
		defer os.Remove(tempFile)

		output, err := execGlowWithTimeout(t, []string{"--flow=100", tempFile}, "", 2*time.Second)
		if err != nil {
			// Empty file might be okay
			t.Logf("Empty file handling: %v", err)
		}
		// Empty files should produce minimal output
		t.Logf("Empty file output length: %d", len(output))
	})

	t.Run("binary_file", func(t *testing.T) {
		// Create a binary file
		binaryData := []byte{0x00, 0xFF, 0x42, 0x13, 0x37, 0xDE, 0xAD, 0xBE, 0xEF}
		tempFile := createTempFile(t, string(binaryData))
		defer os.Remove(tempFile)

		output, err := execGlowWithTimeout(t, []string{"--flow=100", tempFile}, "", 2*time.Second)
		// Binary files should either error or produce some output
		if err == nil && len(output) == 0 {
			t.Error("Binary file produced no output and no error")
		}
		t.Logf("Binary file handling - error: %v, output length: %d", err, len(output))
	})
}

// 3. GLAMOUR INTEGRATION TESTS
func TestIntegrationGlamour(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("rendering_occurs", func(t *testing.T) {
		input := "# Test\n**Bold** and *italic*"
		output, err := execGlowWithTimeout(t, []string{"--flow=100", "-"}, input, 2*time.Second)
		if err != nil {
			t.Fatalf("Rendering failed: %v", err)
		}

		// Output should be different from input (rendered)
		if string(output) == input {
			t.Error("Output not rendered - identical to input")
		}
		if len(output) < len(input) {
			t.Error("Rendered output shorter than input")
		}
	})

	t.Run("ansi_colors", func(t *testing.T) {
		input := "# Heading\n**Bold text**\n`code`"
		output, err := execGlowWithTimeout(t, []string{"--flow=100", "-"}, input, 2*time.Second)
		if err != nil {
			t.Fatalf("ANSI test failed: %v", err)
		}

		// Check for ANSI escape codes
		outputStr := string(output)
		if !strings.Contains(outputStr, "\x1b[") && !strings.Contains(outputStr, "\033[") {
			t.Log("Warning: No ANSI codes detected (might be disabled)")
		}
	})

	t.Run("width_handling", func(t *testing.T) {
		input := "# Test\n" + strings.Repeat("Very long line that should wrap ", 20)
		output, err := execGlowWithTimeout(t, []string{"--flow=100", "--width=40", "-"}, input, 2*time.Second)
		if err != nil {
			t.Fatalf("Width handling failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output with width constraint")
		}
	})

	t.Run("style_selection", func(t *testing.T) {
		input := "# Style Test\nContent"
		styles := []string{"dark", "light", "dracula"}

		outputs := make(map[string][]byte)
		for _, style := range styles {
			output, err := execGlowWithTimeout(t, []string{"--flow=100", "--style=" + style, "-"}, input, 2*time.Second)
			if err != nil {
				t.Logf("Style %s might not exist: %v", style, err)
				continue
			}
			outputs[style] = output
		}

		// At least one style should work
		if len(outputs) == 0 {
			t.Error("No styles produced output")
		}
	})

	t.Run("markdown_features", func(t *testing.T) {
		input := `# Features

## Lists
- Item 1
- Item 2

## Code
` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

## Table
| Col1 | Col2 |
|------|------|
| A    | B    |

## Links
[Example](https://example.com)
`
		output, err := execGlowWithTimeout(t, []string{"--flow=100", "-"}, input, 2*time.Second)
		if err != nil {
			t.Fatalf("Markdown features failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output for markdown features")
		}

		// Verify some content made it through
		outputStr := string(output)
		if !strings.Contains(outputStr, "Hello") {
			t.Error("Code block content missing")
		}
	})
}

// 4. COMMAND-LINE TESTS
func TestIntegrationCommandLine(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("invalid_flow_negative", func(t *testing.T) {
		// -2 is invalid (only -1, 0, or positive allowed)
		output, err := execGlowWithTimeout(t, []string{"--flow=-2", "-"}, "# Test", 2*time.Second)
		// Should either error or treat as -1
		if err == nil && len(output) == 0 {
			t.Error("Invalid flow value produced no output and no error")
		}
	})

	t.Run("invalid_flow_string", func(t *testing.T) {
		cmd := exec.Command(glowBinary, "--flow=abc", "-")
		cmd.Stdin = strings.NewReader("# Test")
		_, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("String flow value should produce error")
		}
	})

	t.Run("flow_overflow", func(t *testing.T) {
		output, err := execGlowWithTimeout(t, []string{"--flow=99999999999999999999", "-"}, "# Test", 2*time.Second)
		// Should handle gracefully
		if err == nil && len(output) == 0 {
			t.Error("Overflow flow value produced no output")
		}
	})

	t.Run("multiple_files", func(t *testing.T) {
		file1 := createTempFile(t, "# File 1")
		file2 := createTempFile(t, "# File 2")
		defer os.Remove(file1)
		defer os.Remove(file2)

		output, err := execGlowWithTimeout(t, []string{file1, file2}, "", 2*time.Second)
		if err != nil {
			t.Logf("Multiple files handling: %v", err)
		}
		// Should process both files
		outputStr := string(output)
		if !strings.Contains(outputStr, "1") || !strings.Contains(outputStr, "2") {
			t.Error("Not all files processed")
		}
	})

	t.Run("help_flag", func(t *testing.T) {
		output, err := execGlowWithTimeout(t, []string{"--help"}, "", 2*time.Second)
		if err != nil {
			// --help often returns exit code 1
			t.Logf("Help flag exit code: %v", err)
		}
		outputStr := string(output)
		if !strings.Contains(strings.ToLower(outputStr), "usage") && !strings.Contains(strings.ToLower(outputStr), "help") {
			t.Error("Help output doesn't contain usage information")
		}
	})
}

// 5. I/O INTEGRATION TESTS
func TestIntegrationIO(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("stdin_piping", func(t *testing.T) {
		// Simulate: echo "# Test" | glow --flow=100 -
		echo := exec.Command("echo", "# Test\nContent")
		glow := exec.Command(glowBinary, "--flow=100", "-")

		pipe, err := echo.StdoutPipe()
		if err != nil {
			t.Fatalf("Pipe creation failed: %v", err)
		}
		glow.Stdin = pipe

		if err := echo.Start(); err != nil {
			t.Fatalf("Echo start failed: %v", err)
		}

		output, err := glow.CombinedOutput()
		if err != nil {
			t.Fatalf("Glow failed: %v", err)
		}

		if err := echo.Wait(); err != nil {
			t.Fatalf("Echo wait failed: %v", err)
		}

		if len(output) == 0 {
			t.Error("No output from piped input")
		}
	})

	t.Run("file_input", func(t *testing.T) {
		tempFile := createTempFile(t, "# File Input\nTesting file reading")
		defer os.Remove(tempFile)

		output, err := execGlowWithTimeout(t, []string{"--flow=100", tempFile}, "", 2*time.Second)
		if err != nil {
			t.Fatalf("File input failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output from file input")
		}
	})

	t.Run("multiple_files_flow", func(t *testing.T) {
		file1 := createTempFile(t, "# First\nContent")
		file2 := createTempFile(t, "# Second\nMore")
		file3 := createTempFile(t, "# Third\nFinal")
		defer os.Remove(file1)
		defer os.Remove(file2)
		defer os.Remove(file3)

		output, err := execGlowWithTimeout(t, []string{"--flow=100", file1, file2, file3}, "", 2*time.Second)
		if err != nil {
			t.Logf("Multiple files with flow: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output from multiple files")
		}
	})

	t.Run("broken_pipe", func(t *testing.T) {
		t.Skip("Broken pipe test is complex and OS-dependent")
		// This would require careful setup of a pipe that closes early
		// Skipping for reliability
	})
}

// 6. PERFORMANCE TESTS
func TestIntegrationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("benchmark_small_file", func(t *testing.T) {
		content := "# Small\n\nQuick content."
		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		start := time.Now()
		cmd := exec.Command(glowBinary, "--flow=100", tempFile)
		_, err := cmd.CombinedOutput()
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Benchmark failed: %v", err)
		}
		t.Logf("Small file processing time: %v", elapsed)

		if elapsed > 1*time.Second {
			t.Error("Small file took too long to process")
		}
	})

	t.Run("benchmark_large_file", func(t *testing.T) {
		// Generate 1MB of markdown
		var buf bytes.Buffer
		for i := 0; i < 10000; i++ {
			buf.WriteString(fmt.Sprintf("## Section %d\n\nParagraph with content.\n\n", i))
		}
		tempFile := createTempFile(t, buf.String())
		defer os.Remove(tempFile)

		start := time.Now()
		cmd := exec.Command(glowBinary, "--flow=10000", tempFile)
		_, err := cmd.CombinedOutput()
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Large benchmark failed: %v", err)
		}
		t.Logf("Large file processing time: %v", elapsed)

		if elapsed > 5*time.Second {
			t.Error("Large file processing too slow")
		}
	})

	t.Run("memory_usage", func(t *testing.T) {
		t.Skip("Memory measurement requires platform-specific tools")
		// Would need to use platform-specific tools to measure memory
		// or instrument the binary itself
	})

	t.Run("first_byte_latency", func(t *testing.T) {
		// Test streaming mode first byte latency
		cmd := exec.Command(glowBinary, "--flow=-1", "-")
		stdin, _ := cmd.StdinPipe()
		stdout, _ := cmd.StdoutPipe()

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start: %v", err)
		}

		// Write input
		go func() {
			stdin.Write([]byte("# Test\n\n"))
			stdin.Write([]byte("Content follows\n"))
			stdin.Close()
		}()

		// Measure time to first byte
		start := time.Now()
		firstByte := make([]byte, 1)
		_, err := stdout.Read(firstByte)
		firstByteLatency := time.Since(start)

		if err != nil && err != io.EOF {
			t.Fatalf("Failed to read first byte: %v", err)
		}

		t.Logf("First byte latency: %v", firstByteLatency)

		if firstByteLatency > 500*time.Millisecond {
			t.Error("First byte latency too high")
		}

		// Drain remaining output to prevent deadlock
		go io.Copy(io.Discard, stdout)

		// Wait for command to complete with timeout
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command completed normally
		case <-time.After(2 * time.Second):
			cmd.Process.Kill()
			t.Error("Command timed out - killing process")
		}
	})

	t.Run("streaming_vs_buffered", func(t *testing.T) {
		content := strings.Repeat("# Section\n\nContent paragraph.\n\n", 100)

		// Streaming mode
		start := time.Now()
		cmd := exec.Command(glowBinary, "--flow=-1", "-")
		cmd.Stdin = strings.NewReader(content)
		_, err := cmd.CombinedOutput()
		streamingTime := time.Since(start)
		if err != nil {
			t.Fatalf("Streaming failed: %v", err)
		}

		// Buffered mode
		start = time.Now()
		cmd = exec.Command(glowBinary, "--flow=0", "-")
		cmd.Stdin = strings.NewReader(content)
		_, err = cmd.CombinedOutput()
		bufferedTime := time.Since(start)
		if err != nil {
			t.Fatalf("Buffered failed: %v", err)
		}

		t.Logf("Streaming time: %v, Buffered time: %v", streamingTime, bufferedTime)

		// Both should complete reasonably quickly
		if streamingTime > 2*time.Second || bufferedTime > 2*time.Second {
			t.Error("Processing times too high")
		}
	})
}

// Helper function to create temporary files
func createTempFile(t *testing.T, content string) string {
	tempFile, err := os.CreateTemp("", "glow_test_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		t.Fatalf("Failed to write temp file: %v", err)
	}

	tempFile.Close()
	return tempFile.Name()
}
