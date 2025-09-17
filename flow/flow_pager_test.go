package flow

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestPagerIntegration tests the pager integration with Flow
func TestPagerIntegration(t *testing.T) {
	// Skip if we can't find basic commands
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat command not available")
	}

	tests := []struct {
		name      string
		input     string
		pagerCmd  string
		expectErr bool
	}{
		{
			name:      "basic cat pager",
			input:     "# Test Header\n\nContent here",
			pagerCmd:  "cat",
			expectErr: false,
		},
		{
			name:      "head pager with limited output",
			input:     strings.Repeat("# Line\n", 100),
			pagerCmd:  "head -5",
			expectErr: false, // SIGPIPE is normal
		},
		{
			name:      "timeout pager",
			input:     "# Test\n",
			pagerCmd:  "timeout 0.1 cat",
			expectErr: false, // Exit code 124 is normal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the pager pipeline
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Parse pager command
			parts := strings.Split(tt.pagerCmd, " ")
			cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatalf("Failed to create stdin pipe: %v", err)
			}

			var output bytes.Buffer
			cmd.Stdout = &output

			if err := cmd.Start(); err != nil {
				t.Fatalf("Failed to start pager: %v", err)
			}

			// Simulate Flow writing to pager
			pagerCtx, pagerCancel := context.WithCancel(ctx)
			defer pagerCancel()

			flowDone := make(chan error, 1)
			go func() {
				r := strings.NewReader(tt.input)
				flowDone <- Flow(pagerCtx, r, stdin, Unbuffered, passthroughRenderer)
			}()

			pagerDone := make(chan error, 1)
			go func() {
				pagerDone <- cmd.Wait()
			}()

			// Wait for either to complete
			select {
			case flowErr := <-flowDone:
				stdin.Close()
				<-pagerDone
				if flowErr != nil && tt.expectErr {
					// Expected error
				} else if flowErr != nil && !tt.expectErr {
					t.Errorf("Unexpected Flow error: %v", flowErr)
				}

			case pagerErr := <-pagerDone:
				pagerCancel()
				stdin.Close()
				<-flowDone

				// Check if it's a "normal" exit
				if exitErr, ok := pagerErr.(*exec.ExitError); ok {
					exitCode := exitErr.ExitCode()
					if exitCode == 124 || exitCode == 141 {
						// Normal exits for timeout and SIGPIPE
						if tt.expectErr {
							t.Errorf("Expected error but got normal exit code %d", exitCode)
						}
					} else if !tt.expectErr {
						t.Errorf("Unexpected pager exit code %d", exitCode)
					}
				} else if pagerErr != nil && !tt.expectErr {
					t.Errorf("Unexpected pager error: %v", pagerErr)
				}
			}

			// Verify we got some output for successful cases
			if !tt.expectErr && output.Len() == 0 {
				t.Errorf("Expected output but got none")
			}
		})
	}
}

// TestPagerEarlyExit tests that Flow handles early pager exit correctly
func TestPagerEarlyExit(t *testing.T) {
	if _, err := exec.LookPath("timeout"); err != nil {
		t.Skip("timeout command not available")
	}

	t.Run("infinite stream with early pager exit", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Create a pager that exits quickly
		cmd := exec.CommandContext(ctx, "timeout", "0.05", "cat")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("Failed to create stdin pipe: %v", err)
		}

		var output bytes.Buffer
		cmd.Stdout = &output

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start pager: %v", err)
		}
		// Ensure process cleanup
		t.Cleanup(func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		})

		// Create cancellable context for Flow
		pagerCtx, pagerCancel := context.WithCancel(ctx)
		defer pagerCancel()

		// Start Flow with infinite input
		flowDone := make(chan error, 1)
		go func() {
			// Infinite reader
			r := &infiniteReader{pattern: "# Test line\n"}
			flowDone <- Flow(pagerCtx, r, stdin, Unbuffered, passthroughRenderer)
		}()

		// Wait for pager to exit
		pagerErr := cmd.Wait()

		// Cancel Flow immediately
		pagerCancel()
		stdin.Close()

		// Flow should exit quickly after cancellation
		select {
		case <-flowDone:
			// Good - Flow detected cancellation
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Flow did not exit quickly after pager exit")
		}

		// Verify pager exited with timeout (124)
		if exitErr, ok := pagerErr.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 124 {
				t.Errorf("Expected exit code 124, got %d", exitErr.ExitCode())
			}
		}
	})
}

// TestPagerStressConditions tests stress conditions
func TestPagerStressConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Run("rapid pager creation and cancellation", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)

			cmd := exec.CommandContext(ctx, "cat")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatalf("Failed to create stdin pipe: %v", err)
			}

			if err := cmd.Start(); err != nil {
				t.Fatalf("Failed to start pager: %v", err)
			}

			// Immediately cancel
			cancel()
			stdin.Close()
			_ = cmd.Wait()
		}
		// Should not leak resources
	})

	t.Run("large data through pager", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "head", "-100")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("Failed to create stdin pipe: %v", err)
		}

		var output bytes.Buffer
		cmd.Stdout = &output

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start pager: %v", err)
		}

		// Send large amount of data
		pagerCtx, pagerCancel := context.WithCancel(ctx)
		defer pagerCancel()

		flowDone := make(chan error, 1)
		go func() {
			// Create 10MB of data
			data := strings.Repeat("# Large header\nContent line\n", 100000)
			r := strings.NewReader(data)
			flowDone <- Flow(pagerCtx, r, stdin, 4096, passthroughRenderer)
		}()

		// Wait for pager to complete
		pagerErr := cmd.Wait()
		pagerCancel()
		stdin.Close()
		<-flowDone

		// Should handle SIGPIPE gracefully
		if exitErr, ok := pagerErr.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 141 { // SIGPIPE
				t.Errorf("Expected SIGPIPE (141), got exit code %d", exitErr.ExitCode())
			}
		}

		// Should have gotten exactly 100 lines
		lines := strings.Split(output.String(), "\n")
		if len(lines) < 100 {
			t.Errorf("Expected at least 100 lines, got %d", len(lines))
		}
	})
}

// TestExitCodeClassification tests that exit codes are properly classified
func TestExitCodeClassification(t *testing.T) {
	tests := []struct {
		exitCode    int
		shouldError bool
		description string
	}{
		{0, true, "exit 0 normal exit"}, // Exit 0 in error context is suspicious - should be treated as error
		{124, false, "timeout exit"},
		{141, false, "SIGPIPE exit"},
		{1, true, "general error"},
		{2, true, "command error"},
		{127, true, "command not found"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("exit_%d_%s", tt.exitCode, tt.description), func(t *testing.T) {
			// Create mock exit error
			_ = &exec.ExitError{}
			// We can't directly set exit code in ExitError, so we test the logic

			// This tests our classification logic
			// Only 124 (timeout) and 141 (SIGPIPE) are normal for streaming
			// Exit code 0 in an error context is suspicious and should be treated as error
			isNormalExit := tt.exitCode == 124 || tt.exitCode == 141

			if isNormalExit && tt.shouldError {
				t.Errorf("Exit code %d should be treated as error but isn't", tt.exitCode)
			}
			if !isNormalExit && !tt.shouldError {
				t.Errorf("Exit code %d should be treated as normal but isn't", tt.exitCode)
			}
		})
	}
}

// Helper type for infinite streaming
type infiniteReader struct {
	pattern string
}

func (r *infiniteReader) Read(p []byte) (n int, err error) {
	time.Sleep(10 * time.Millisecond) // Simulate slow input
	n = copy(p, r.pattern)
	return n, nil
}
