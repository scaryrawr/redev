package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	devssh "github.com/scaryrawr/devssh"
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
	for _, unwanted := range []string{
		"-l forward-github-token",
		"-l no-forward-github-token",
		"_devpod-stdio-proxy",
		"-l plain-devpod-ssh",
		"-l no-port-monitor",
		"-l no-xdg-open",
		"-l no-browser",
		"-l no-notifications",
		"-l install-xdg-open",
	} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("completion includes unwanted flag %q:\n%s", unwanted, output)
		}
	}
}

func TestRunSSHUsesDevSSHWithDevpodProxy(t *testing.T) {
	var stdout, stderr bytes.Buffer
	originalRunDevSSH := runDevSSH
	originalExecutablePath := executablePath
	originalGitHubAuthToken := githubAuthToken
	originalDevpodSSHUser := devpodSSHUser
	t.Cleanup(func() {
		runDevSSH = originalRunDevSSH
		executablePath = originalExecutablePath
		githubAuthToken = originalGitHubAuthToken
		devpodSSHUser = originalDevpodSSHUser
	})
	t.Setenv("DEVSSH_CONFIG", filepath.Join(t.TempDir(), "missing-config.json"))
	t.Setenv("TERM", "xterm-256color")

	githubAuthToken = func(context.Context) (string, error) {
		t.Fatal("token should be resolved by the ProxyCommand helper, not before devssh starts")
		return "", nil
	}
	executablePath = func() (string, error) {
		return "/tmp/redev binary", nil
	}
	devpodSSHUser = func(workspace string) (string, error) {
		if workspace != "my-workspace" {
			t.Fatalf("devpodSSHUser workspace = %q, want my-workspace", workspace)
		}
		return "vscode", nil
	}

	var got devssh.Options
	runDevSSH = func(ctx context.Context, opts devssh.Options) error {
		got = opts
		return nil
	}

	err := Run(context.Background(), []string{
		"ssh",
		"my-workspace",
		"--",
		"-L",
		"3000:localhost:3000",
		"htop",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.HasPrefix(got.Host, "redev-devpod-my-workspace-") {
		t.Fatalf("Host = %q, want sanitized devpod alias", got.Host)
	}
	wantSSHArgs := []string{"-L", "3000:localhost:3000", "htop"}
	if !reflect.DeepEqual(got.SSHArgs, wantSSHArgs) {
		t.Fatalf("SSHArgs = %#v, want %#v", got.SSHArgs, wantSSHArgs)
	}
	if !got.DisableDefaultReversePortForwards || len(got.ReversePortForwards) == 0 {
		t.Fatalf("expected merged reverse forwards without implicit defaults: %+v", got)
	}
	wantSSHOptionsPrefix := []string{
		"-o", "User=vscode",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o",
	}
	if len(got.SSHOptions) != len(wantSSHOptionsPrefix)+1 ||
		!reflect.DeepEqual(got.SSHOptions[:len(wantSSHOptionsPrefix)], wantSSHOptionsPrefix) {
		t.Fatalf("SSHOptions = %#v, want configured user, host-key bypass, and ProxyCommand", got.SSHOptions)
	}
	proxyCommand := strings.TrimPrefix(got.SSHOptions[len(got.SSHOptions)-1], "ProxyCommand=")
	for _, want := range []string{"'/tmp/redev binary'", "_devpod-stdio-proxy", "--user", "vscode", "my-workspace"} {
		if !strings.Contains(proxyCommand, want) {
			t.Fatalf("ProxyCommand = %q, missing %q", proxyCommand, want)
		}
	}
	if strings.Contains(proxyCommand, "root") {
		t.Fatalf("ProxyCommand forced root user: %q", proxyCommand)
	}
	if strings.Contains(proxyCommand, "secret-token") {
		t.Fatalf("ProxyCommand leaked token: %q", proxyCommand)
	}
}

func TestRunSSHUsesGhosttyTermFallback(t *testing.T) {
	var stdout, stderr bytes.Buffer
	originalRunDevSSH := runDevSSH
	originalExecutablePath := executablePath
	originalDevpodSSHUser := devpodSSHUser
	t.Cleanup(func() {
		runDevSSH = originalRunDevSSH
		executablePath = originalExecutablePath
		devpodSSHUser = originalDevpodSSHUser
	})
	t.Setenv("DEVSSH_CONFIG", filepath.Join(t.TempDir(), "missing-config.json"))
	t.Setenv("TERM", "xterm-ghostty")

	executablePath = func() (string, error) {
		return "/tmp/redev", nil
	}
	devpodSSHUser = func(workspace string) (string, error) {
		return "vscode", nil
	}

	var got devssh.Options
	runDevSSH = func(ctx context.Context, opts devssh.Options) error {
		got = opts
		return nil
	}

	if err := Run(context.Background(), []string{"ssh", "my-workspace"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !containsOptionPair(got.SSHOptions, "-o", "SetEnv=TERM=xterm-256color") {
		t.Fatalf("SSHOptions = %#v, want Ghostty TERM fallback", got.SSHOptions)
	}
}

func TestRunSSHDefaultsEnableDevSSHFeatures(t *testing.T) {
	var stdout, stderr bytes.Buffer
	originalRunDevSSH := runDevSSH
	originalExecutablePath := executablePath
	originalDevpodSSHUser := devpodSSHUser
	t.Cleanup(func() {
		runDevSSH = originalRunDevSSH
		executablePath = originalExecutablePath
		devpodSSHUser = originalDevpodSSHUser
	})
	t.Setenv("DEVSSH_CONFIG", filepath.Join(t.TempDir(), "missing-config.json"))

	executablePath = func() (string, error) {
		return "/tmp/redev", nil
	}
	devpodSSHUser = func(workspace string) (string, error) {
		return "vscode", nil
	}

	var got devssh.Options
	runDevSSH = func(ctx context.Context, opts devssh.Options) error {
		got = opts
		return nil
	}

	if err := Run(context.Background(), []string{"ssh", "my-workspace"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got.DisablePortMonitor {
		t.Fatal("port monitoring should be enabled by default")
	}
	if got.DisableXdgOpen {
		t.Fatal("xdg-open shim upload/install should be enabled by default")
	}
	if got.DisableBrowser {
		t.Fatal("browser opener should be enabled by default")
	}
	if got.DisableNotifications {
		t.Fatal("notifications should be enabled by default")
	}
}

func TestDevpodStdioProxyForwardsGitHubTokenAsEnvironment(t *testing.T) {
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

	if err := Run(context.Background(), []string{"_devpod-stdio-proxy", "--user", "vscode", "my-workspace"}, &stdout, &stderr); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	wantArgs := []string{"ssh", "--stdio", "--start-services=false", "--send-env", "GH_TOKEN", "--user", "vscode", "my-workspace"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("devpod args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !containsArg(gotArgs, "--start-services=false") {
		t.Fatalf("devpod stdio proxy must disable DevPod services to avoid duplicate forwarding: %#v", gotArgs)
	}
	if strings.Contains(strings.Join(gotArgs, " "), "secret-token") {
		t.Fatalf("devpod args leaked token: %#v", gotArgs)
	}

	wantEnv := []string{"GH_TOKEN=secret-token"}
	if !reflect.DeepEqual(gotEnv, wantEnv) {
		t.Fatalf("devpod env = %#v, want %#v", gotEnv, wantEnv)
	}
}

func TestDevpodProxyCommandEscapesOpenSSHPercentTokens(t *testing.T) {
	got := devpodProxyCommand("/tmp/redev", "space %h %p", "vs%r")
	for _, want := range []string{"%%h", "%%p"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ProxyCommand = %q, missing escaped token %q", got, want)
		}
	}
	withoutEscapedTokens := strings.ReplaceAll(strings.ReplaceAll(got, "%%h", ""), "%%p", "")
	if strings.Contains(withoutEscapedTokens, "%h") || strings.Contains(withoutEscapedTokens, "%p") {
		t.Fatalf("ProxyCommand contains unescaped OpenSSH percent tokens: %q", got)
	}
	if !strings.Contains(got, "vs%%r") {
		t.Fatalf("ProxyCommand did not escape user token: %q", got)
	}
}

func TestDevpodStdioProxyReturnsTokenError(t *testing.T) {
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

	err := Run(context.Background(), []string{"_devpod-stdio-proxy", "my-workspace"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("Run returned nil error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q, want token error", err.Error())
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func containsOptionPair(args []string, option, value string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == option && args[i+1] == value {
			return true
		}
	}
	return false
}
