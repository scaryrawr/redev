// Package cli implements redev's command-line interface.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/scaryrawr/redev/internal/devpod"
)

const version = "0.1.0-dev"

var (
	runDevpodInteractiveWithEnv = devpod.RunInteractiveWithEnv
	githubAuthToken             = currentGitHubAuthToken
)

// Run parses args and executes the requested command.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	case "version":
		fmt.Fprintf(stdout, "redev %s\n", version)
		return nil
	case "doctor":
		return runDoctor(ctx, args[1:], stdout, stderr)
	case "list":
		return runList(ctx, args[1:], stdout, stderr)
	case "open":
		return runOpen(ctx, args[1:], stdout, stderr)
	case "ssh":
		return runSSH(ctx, args[1:], stdout, stderr)
	case "completion":
		return runCompletion(args[1:], stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runDoctor(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("doctor", stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("doctor does not accept positional arguments")
	}

	checks := []struct {
		name        string
		versionArgs []string
		err         error
	}{
		{name: "devpod", versionArgs: []string{"version"}},
		{name: "ssh", versionArgs: []string{"-V"}},
		{name: "gh", versionArgs: []string{"--version"}},
	}

	ok := true
	for _, check := range checks {
		err := requireCommand(ctx, check.name, check.versionArgs...)
		if err != nil {
			ok = false
			fmt.Fprintf(stdout, "missing %s: %v\n", check.name, err)
			continue
		}
		fmt.Fprintf(stdout, "ok %s\n", check.name)
	}
	if !ok {
		return errors.New("one or more required tools are unavailable")
	}
	return nil
}

func runList(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("list", stderr)
	jsonOutput := fs.Bool("json", false, "print devpod JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("list does not accept positional arguments")
	}

	devpodArgs := []string{"list"}
	if *jsonOutput {
		devpodArgs = append(devpodArgs, "--output", "json")
	}
	return devpod.Run(ctx, stdout, stderr, devpodArgs...)
}

func runOpen(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("open", stderr)
	ide := fs.String("ide", "", "IDE to pass to devpod")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("open requires exactly one workspace")
	}

	devpodArgs := []string{"open", fs.Arg(0)}
	if *ide != "" {
		devpodArgs = append(devpodArgs, "--ide", *ide)
	}
	return devpod.Run(ctx, stdout, stderr, devpodArgs...)
}

func runSSH(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("ssh", stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("ssh requires a workspace")
	}

	devpodArgs := []string{"ssh"}
	var env []string
	token, err := githubAuthToken(ctx)
	if err != nil {
		return fmt.Errorf("forward GitHub token: %w", err)
	}
	env = append(env, "GH_TOKEN="+token)
	devpodArgs = append(devpodArgs, "--send-env", "GH_TOKEN")

	devpodArgs = append(devpodArgs, fs.Arg(0))
	devpodArgs = append(devpodArgs, fs.Args()[1:]...)
	return runDevpodInteractiveWithEnv(ctx, os.Stdin, stdout, stderr, env, devpodArgs...)
}

func runCompletion(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("completion requires a shell")
	}
	if args[0] != "fish" {
		return fmt.Errorf("unsupported shell %q", args[0])
	}
	fmt.Fprint(stdout, fishCompletion())
	return nil
}

func requireCommand(ctx context.Context, name string, versionArgs ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, path, versionArgs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s found at %s but failed version check: %w", name, path, err)
	}
	return nil
}

func currentGitHubAuthToken(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("get GitHub auth token with gh: %w", err)
	}
	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", errors.New("get GitHub auth token with gh: empty token")
	}
	return token, nil
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func printUsage(w io.Writer) {
	commands := []string{"completion fish", "doctor", "list", "open <workspace>", "ssh <workspace> [-- ssh-args...]"}
	sort.Strings(commands)

	fmt.Fprintln(w, "redev is a devpod-oriented remote development helper.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  redev <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, command := range commands {
		fmt.Fprintf(w, "  %s\n", command)
	}
}

func fishCompletion() string {
	commands := []string{
		"completion\tGenerate shell completions",
		"doctor\tCheck required local tools",
		"help\tShow help",
		"list\tList devpod workspaces",
		"open\tOpen a devpod workspace",
		"ssh\tStart a shell in a devpod workspace",
		"version\tPrint redev version",
	}

	var b strings.Builder
	b.WriteString("# fish completion for redev\n")
	b.WriteString("complete -c redev -e\n")
	for _, command := range commands {
		name, description, _ := strings.Cut(command, "\t")
		fmt.Fprintf(&b, "complete -c redev -f -n '__fish_use_subcommand' -a %q -d %q\n", name, description)
	}
	b.WriteString("complete -c redev -f -n '__fish_seen_subcommand_from completion' -a fish -d 'Generate fish completions'\n")
	b.WriteString("complete -c redev -f -n '__fish_seen_subcommand_from open' -l ide -d 'IDE to pass to devpod' -r\n")
	b.WriteString("complete -c redev -f -n '__fish_seen_subcommand_from list' -l json -d 'Print devpod JSON output'\n")
	return b.String()
}
