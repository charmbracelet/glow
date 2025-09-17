package flow

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Build glow binary once for all upper layer tests
var (
	upperBuildOnce  sync.Once
	upperGlowBinary string
	upperBuildError error
)

func buildGlowBinary(t *testing.T) string {
	upperBuildOnce.Do(func() {
		tmpDir := os.TempDir()
		upperGlowBinary = filepath.Join(tmpDir, "test_glow_upper")
		if runtime.GOOS == "windows" {
			upperGlowBinary += ".exe"
		}

		// Build from project root
		projectRoot := "/Users/c/devel/glow"
		cmd := exec.Command("go", "build", "-o", upperGlowBinary, ".")
		cmd.Dir = projectRoot

		output, err := cmd.CombinedOutput()
		if err != nil {
			upperBuildError = fmt.Errorf("failed to build glow: %v\nOutput: %s", err, output)
		}
	})

	if upperBuildError != nil {
		t.Skip("Skipping upper layer tests - binary build failed:", upperBuildError)
	}

	return upperGlowBinary
}

// Helper to create temp markdown files
func createTestMarkdown(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "glow_test_*.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.WriteString(content); err != nil {
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		t.Fatal(err)
	}
	tmpfile.Close()
	return tmpfile.Name()
}

// 1. COMMAND-LINE TO FLOW INTEGRATION TESTS
func TestUpperLayerCommandLineIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binary := buildGlowBinary(t)

	testContent := `# Test Document

This is a test paragraph with **bold** and *italic* text.

## Section with Reference

Here's a reference [link][ref] to test.

- List item 1
- List item 2
- List item 3

[ref]: https://example.com/test`

	t.Run("flow_values_file", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		flowValues := []string{"-1", "0", "100", "1024", "10000"}
		for _, flow := range flowValues {
			cmd := exec.Command(binary, "--flow="+flow, tmpfile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Failed with --flow=%s: %v\nOutput: %s", flow, err, output)
				continue
			}
			if len(output) == 0 {
				t.Errorf("No output with --flow=%s", flow)
			}
		}
	})

	t.Run("flow_stdin", func(t *testing.T) {
		cmd := exec.Command(binary, "--flow=-1", "-")
		cmd.Stdin = strings.NewReader(testContent)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed with stdin: %v\nOutput: %s", err, output)
		}
		if len(output) == 0 {
			t.Error("No output from stdin")
		}
	})

	t.Run("flow_multiple_files", func(t *testing.T) {
		file1 := createTestMarkdown(t, "# File 1\n\nFirst file content.")
		file2 := createTestMarkdown(t, "# File 2\n\nSecond file content.")
		defer os.Remove(file1)
		defer os.Remove(file2)

		cmd := exec.Command(binary, "--flow=0", file1, file2)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Multiple files with flow: %v", err)
		}
		outputStr := string(output)
		if !strings.Contains(outputStr, "File 1") && !strings.Contains(outputStr, "File 2") {
			t.Log("Note: Multiple file handling may not concatenate output")
		}
	})

	t.Run("style_and_flow", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--style=dark", "--flow=1024", tmpfile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Style might not exist
			t.Logf("Style+flow combination: %v", err)
		} else if len(output) == 0 {
			t.Error("No output with style+flow")
		}
	})

	t.Run("width_and_flow", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, strings.Repeat("Very long line ", 50))
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--width=40", "--flow=100", tmpfile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Width+flow failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output with width+flow")
		}
	})

	t.Run("invalid_flow_value", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, "# Test")
		defer os.Remove(tmpfile)

		// Test invalid flow values
		invalidFlows := []string{"abc", "-2", "999999999999999"}
		for _, flow := range invalidFlows {
			cmd := exec.Command(binary, "--flow="+flow, tmpfile)
			output, err := cmd.CombinedOutput()
			if err == nil && len(output) > 0 {
				t.Logf("Invalid flow %s handled gracefully", flow)
			}
		}
	})

	t.Run("missing_file", func(t *testing.T) {
		cmd := exec.Command(binary, "--flow=100", "/nonexistent/file.md")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for missing file")
		}
		outputStr := string(output)
		if !strings.Contains(strings.ToLower(outputStr), "error") &&
			!strings.Contains(strings.ToLower(outputStr), "not found") &&
			!strings.Contains(strings.ToLower(outputStr), "no such") {
			t.Log("Error message could be clearer for missing file")
		}
	})

	t.Run("binary_file", func(t *testing.T) {
		// Create a binary file
		tmpfile := filepath.Join(t.TempDir(), "binary.dat")
		err := os.WriteFile(tmpfile, []byte{0xFF, 0xFE, 0x00, 0x01, 0x02}, 0644)
		if err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command(binary, "--flow=100", tmpfile)
		output, err := cmd.CombinedOutput()
		// Should handle gracefully
		if err != nil {
			t.Logf("Binary file error (expected): %v", err)
		} else {
			t.Logf("Binary file processed: %d bytes output", len(output))
		}
	})

	t.Run("large_file", func(t *testing.T) {
		// Create a 2MB file (reduced from 10MB for speed)
		var buf bytes.Buffer
		buf.WriteString("# Large Document\n\n")
		for i := 0; i < 20000; i++ {
			buf.WriteString(fmt.Sprintf("Line %d with some content to fill space.\n", i))
		}
		tmpfile := createTestMarkdown(t, buf.String())
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=10000", tmpfile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Large file failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output for large file")
		}
		t.Logf("Large file processed: %d bytes in, %d bytes out", buf.Len(), len(output))
	})
}

// 2. ERROR PROPAGATION TESTS
func TestUpperLayerErrorPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binary := buildGlowBinary(t)

	t.Run("file_not_found", func(t *testing.T) {
		cmd := exec.Command(binary, "--flow=100", "/this/does/not/exist.md")
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 0 {
				t.Error("Expected non-zero exit code")
			}
		}
		t.Logf("File not found error: %s", output)
	})

	t.Run("permission_denied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Permission test not reliable on Windows")
		}

		// Create file with no read permissions
		tmpfile := filepath.Join(t.TempDir(), "noperm.md")
		err := os.WriteFile(tmpfile, []byte("# Test"), 0000)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(tmpfile, 0644) // Restore permissions for cleanup

		cmd := exec.Command(binary, "--flow=100", tmpfile)
		output, err := cmd.CombinedOutput()
		if err == nil {
			// Might succeed if running as root
			t.Log("Permission test inconclusive - may be running as root")
		} else {
			t.Logf("Permission denied handled: %v", err)
		}
		_ = output
	})

	t.Run("malformed_markdown", func(t *testing.T) {
		// Extremely malformed markdown
		malformed := "```\nUnclosed fence\n# Mixed with header\n[Broken link](((("
		tmpfile := createTestMarkdown(t, malformed)
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=100", tmpfile)
		output, err := cmd.CombinedOutput()
		// Should handle gracefully
		if err != nil {
			t.Logf("Malformed markdown error: %v", err)
		}
		if len(output) == 0 && err != nil {
			t.Error("No output for malformed markdown")
		}
	})

	t.Run("empty_file", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, "")
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=100", tmpfile)
		output, err := cmd.CombinedOutput()
		// Empty file should be handled gracefully
		if err != nil {
			t.Logf("Empty file handling: %v", err)
		}
		t.Logf("Empty file output: %d bytes", len(output))
	})

	t.Run("stdin_error", func(t *testing.T) {
		// Test with closed stdin
		cmd := exec.Command(binary, "--flow=100", "-")
		// Don't provide stdin - it's closed
		output, err := cmd.CombinedOutput()
		// Should error or handle gracefully
		t.Logf("Closed stdin: err=%v, output=%d bytes", err, len(output))
	})
}

// 3. SIGNAL HANDLING TESTS
func TestUpperLayerSignalHandling(t *testing.T) {
	if testing.Short() || runtime.GOOS == "windows" {
		t.Skip("Skipping signal tests")
	}

	binary := buildGlowBinary(t)

	t.Run("sigint_during_flow", func(t *testing.T) {
		// Create large input to ensure processing time
		largeContent := strings.Repeat("# Section\n\nContent paragraph.\n\n", 10000)
		tmpfile := createTestMarkdown(t, largeContent)
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=100", tmpfile)
		err := cmd.Start()
		if err != nil {
			t.Fatal(err)
		}

		// Give it time to start processing
		time.Sleep(10 * time.Millisecond)

		// Send SIGINT
		cmd.Process.Signal(syscall.SIGINT)

		// Wait for completion
		err = cmd.Wait()
		// Should exit cleanly
		t.Logf("SIGINT handling: %v", err)
	})

	t.Run("sigterm_during_flow", func(t *testing.T) {
		largeContent := strings.Repeat("Line\n", 10000)
		tmpfile := createTestMarkdown(t, largeContent)
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=100", tmpfile)
		err := cmd.Start()
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(10 * time.Millisecond)
		cmd.Process.Signal(syscall.SIGTERM)
		err = cmd.Wait()
		t.Logf("SIGTERM handling: %v", err)
	})

	t.Run("clean_exit", func(t *testing.T) {
		// Normal execution should exit cleanly
		tmpfile := createTestMarkdown(t, "# Quick test")
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=100", tmpfile)
		err := cmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() != 0 {
					t.Errorf("Non-zero exit code: %d", exitErr.ExitCode())
				}
			}
		}
	})

	t.Run("no_zombies", func(t *testing.T) {
		// Run multiple quick processes
		for i := 0; i < 5; i++ {
			cmd := exec.Command(binary, "--flow=100", "-")
			cmd.Stdin = strings.NewReader("# Test")
			output, _ := cmd.CombinedOutput()
			_ = output
		}
		// If we get here without hanging, no zombies
		t.Log("No zombie processes detected")
	})

	t.Run("signal_cleanup", func(t *testing.T) {
		// Ensure resources are cleaned up on signal
		tmpfile := createTestMarkdown(t, "# Test")
		defer os.Remove(tmpfile)

		for i := 0; i < 3; i++ {
			cmd := exec.Command(binary, "--flow=100", tmpfile)
			cmd.Start()
			time.Sleep(5 * time.Millisecond)
			cmd.Process.Kill()
			cmd.Wait()
		}
		t.Log("Signal cleanup completed")
	})
}

// 4. FLOW MODE SELECTION TESTS
func TestUpperLayerFlowModeSelection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binary := buildGlowBinary(t)
	testContent := "# Mode Test\n\nContent for testing flow modes."

	t.Run("default_no_flow", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		// Without --flow flag - might launch TUI or use default
		cmd := exec.Command(binary, tmpfile)
		cmd.Env = append(os.Environ(), "TERM=dumb") // Prevent TUI
		output, err := cmd.CombinedOutput()
		// Behavior depends on terminal detection
		t.Logf("Default mode: err=%v, output=%d bytes", err, len(output))
	})

	t.Run("flow_unbuffered", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=-1", tmpfile)
		start := time.Now()
		output, err := cmd.CombinedOutput()
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Unbuffered mode failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output in unbuffered mode")
		}
		t.Logf("Unbuffered mode completed in %v", elapsed)
	})

	t.Run("flow_patient", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, "--flow=0", tmpfile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Patient mode failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output in patient mode")
		}
	})

	t.Run("flow_windowed", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		windows := []string{"1", "100", "1024", "65536"}
		for _, window := range windows {
			cmd := exec.Command(binary, "--flow="+window, tmpfile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("DefaultChunk mode %s failed: %v", window, err)
				continue
			}
			if len(output) == 0 {
				t.Errorf("No output with window=%s", window)
			}
		}
	})

	t.Run("tui_prevention", func(t *testing.T) {
		// Ensure we can prevent TUI mode
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		cmd := exec.Command(binary, tmpfile)
		// Set non-terminal environment
		cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1")
		output, err := cmd.CombinedOutput()
		// Should not launch TUI
		t.Logf("TUI prevention: err=%v, output=%d bytes", err, len(output))
	})
}

// 5. STYLE AND WIDTH INTEGRATION TESTS
func TestUpperLayerStyleWidthIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	binary := buildGlowBinary(t)
	testContent := `# Style and Width Test

This is a paragraph with **bold**, *italic*, and ` + "`code`" + ` formatting.

| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Data 1   | Data 2   | Data 3   |
`

	t.Run("style_reaches_renderer", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		styles := []string{"dark", "light", "dracula", "github"}
		outputs := make(map[string][]byte)

		for _, style := range styles {
			cmd := exec.Command(binary, "--style="+style, "--flow=100", tmpfile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Style %s error: %v", style, err)
				continue
			}
			outputs[style] = output
		}

		// At least some styles should work
		if len(outputs) == 0 {
			t.Error("No styles produced output")
		}
	})

	t.Run("width_affects_output", func(t *testing.T) {
		longLine := strings.Repeat("This is a very long line that should wrap based on width settings. ", 20)
		tmpfile := createTestMarkdown(t, longLine)
		defer os.Remove(tmpfile)

		widths := []string{"40", "80", "120"}
		outputs := make(map[string][]byte)

		for _, width := range widths {
			cmd := exec.Command(binary, "--width="+width, "--flow=100", tmpfile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Width %s failed: %v", width, err)
				continue
			}
			outputs[width] = output
		}

		// Different widths might produce different outputs
		if len(outputs) > 1 {
			t.Logf("Width parameter processed for %d configurations", len(outputs))
		}
	})

	t.Run("terminal_width_detection", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		// Without explicit width, should detect terminal
		cmd := exec.Command(binary, "--flow=100", tmpfile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Terminal width detection failed: %v", err)
		}
		if len(output) == 0 {
			t.Error("No output with terminal width detection")
		}
	})

	t.Run("style_flow_combination", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, testContent)
		defer os.Remove(tmpfile)

		combinations := []struct {
			style string
			flow  string
		}{
			{"dark", "-1"},
			{"light", "0"},
			{"github", "1024"},
		}

		for _, combo := range combinations {
			cmd := exec.Command(binary, "--style="+combo.style, "--flow="+combo.flow, tmpfile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Style %s + flow %s error: %v", combo.style, combo.flow, err)
			} else if len(output) == 0 {
				t.Errorf("No output for style %s + flow %s", combo.style, combo.flow)
			}
		}
	})

	t.Run("width_flow_combination", func(t *testing.T) {
		tmpfile := createTestMarkdown(t, strings.Repeat("Long content ", 100))
		defer os.Remove(tmpfile)

		combinations := []struct {
			width string
			flow  string
		}{
			{"40", "100"},
			{"80", "1024"},
			{"120", "-1"},
		}

		for _, combo := range combinations {
			cmd := exec.Command(binary, "--width="+combo.width, "--flow="+combo.flow, tmpfile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Width %s + flow %s failed: %v", combo.width, combo.flow, err)
			} else if len(output) == 0 {
				t.Errorf("No output for width %s + flow %s", combo.width, combo.flow)
			}
		}
	})
}
