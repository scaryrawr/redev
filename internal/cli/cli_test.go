package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestRunHelpPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "redev <command>") {
		t.Fatalf("usage missing command synopsis:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestUnknownCommandReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Run(context.Background(), []string{"wat"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run returned nil error for unknown command")
	}
	if !strings.Contains(err.Error(), `unknown command "wat"`) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestFishCompletionIncludesCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if err := Run(context.Background(), []string{"completion", "fish"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"complete -c redev -e", "-a \"open\"", "-a \"ssh\"", "-l ide"} {
		if !strings.Contains(output, want) {
			t.Fatalf("completion missing %q:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{"-l forward-github-token", "-l no-forward-github-token"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("completion includes unwanted flag %q:\n%s", unwanted, output)
		}
	}
}

func TestRunSSHForwardsGitHubTokenAsEnvironment(t *testing.T) {
	var stdout, stderr bytes.Buffer
	originalRunDevpodInteractiveWithEnv := runDevpodInteractiveWithEnv
	originalGitHubAuthToken := githubAuthToken
	t.Cleanup(func() {
		runDevpodInteractiveWithEnv = originalRunDevpodInteractiveWithEnv
		githubAuthToken = originalGitHubAuthToken
	})

	githubAuthToken = func(context.Context) (string, error) {
		return "secret-token", nil
	}

	var gotEnv []string
	var gotArgs []string
	runDevpodInteractiveWithEnv = func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, env []string, args ...string) error {
		gotEnv = append([]string(nil), env...)
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := Run(context.Background(), []string{"ssh", "my-workspace", "--", "echo", "hi"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	wantArgs := []string{"ssh", "--send-env", "GH_TOKEN", "my-workspace", "--", "echo", "hi"}
	if strings.Join(gotArgs, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("devpod args = %#v, want %#v", gotArgs, wantArgs)
	}
	if strings.Contains(strings.Join(gotArgs, " "), "secret-token") {
		t.Fatalf("devpod args leaked token: %#v", gotArgs)
	}

	wantEnv := []string{"GH_TOKEN=secret-token"}
	if strings.Join(gotEnv, "\x00") != strings.Join(wantEnv, "\x00") {
		t.Fatalf("devpod env = %#v, want %#v", gotEnv, wantEnv)
	}
}

func TestRunSSHAutoForwardGitHubTokenReturnsTokenError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	originalRunDevpodInteractiveWithEnv := runDevpodInteractiveWithEnv
	originalGitHubAuthToken := githubAuthToken
	t.Cleanup(func() {
		runDevpodInteractiveWithEnv = originalRunDevpodInteractiveWithEnv
		githubAuthToken = originalGitHubAuthToken
	})

	githubAuthToken = func(context.Context) (string, error) {
		return "", errors.New("boom")
	}
	runDevpodInteractiveWithEnv = func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, env []string, args ...string) error {
		t.Fatal("devpod should not run when token lookup fails")
		return nil
	}

	err := Run(context.Background(), []string{"ssh", "my-workspace"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q, want token error", err.Error())
	}
}
