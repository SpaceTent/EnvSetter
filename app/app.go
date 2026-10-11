package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/SpaceTent/EnvSetter/pkg/envvaultclient"
)

const usage = `EnvSetter fetches an encrypted .env file from EnvVault.net and decrypts it locally.

Usage:
  envsetter pull [options]
  envsetter run [options] -- command [args...]
  envsetter VAULT SECRET OUTPUT                 legacy pull syntax
  envsetter version

Options:
  --vault, -v NAME       vault URL name (or ENVVAULT_VAULT)
  --secret, -s NAME      secret URL name (or ENVVAULT_SECRET)
  --url URL              API origin (or ENVVAULT_URL; defaults to https://www.envvault.net)
  --token-file PATH      read the vault token from a file instead of ENVVAULT_TOKEN
  --key-file PATH        read the decryption key from a file instead of ENVVAULT_KEY

Pull options:
  --output, -o PATH      destination .env file
  --force                atomically replace an existing destination

The token is sent as an Authorization header. The decryption key never leaves this process.
`

// Run executes the CLI and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	case "version", "--version":
		fmt.Fprintln(stdout, version)
		return 0
	case "pull":
		return runPull(args[1:], stdout, stderr)
	case "run":
		return runCommand(args[1:], stdout, stderr)
	default:
		if len(args) == 3 && !strings.HasPrefix(args[0], "-") {
			return pull(stdout, stderr, commonOptions{
				vault:  normalizeVault(args[0]),
				secret: args[1],
			}, args[2], false)
		}
		fmt.Fprintf(stderr, "envsetter: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

type commonOptions struct {
	url       string
	vault     string
	secret    string
	tokenFile string
	keyFile   string
}

// transport is nil in production. Tests replace it with an in-memory HTTP
// transport so the suite does not need a loopback listener.
var transport http.RoundTripper

func addCommonFlags(fs *flag.FlagSet, options *commonOptions) {
	fs.StringVar(&options.url, "url", "", "EnvVault API origin")
	fs.StringVar(&options.vault, "vault", "", "vault URL name")
	fs.StringVar(&options.vault, "v", "", "vault URL name")
	fs.StringVar(&options.secret, "secret", "", "secret URL name")
	fs.StringVar(&options.secret, "s", "", "secret URL name")
	fs.StringVar(&options.tokenFile, "token-file", "", "file containing the vault token")
	fs.StringVar(&options.keyFile, "key-file", "", "file containing the decryption key")
}

func runPull(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var options commonOptions
	var output string
	var force bool
	addCommonFlags(fs, &options)
	fs.StringVar(&output, "output", "", "destination .env file")
	fs.StringVar(&output, "o", "", "destination .env file")
	fs.BoolVar(&force, "force", false, "replace an existing destination")
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "envsetter: pull does not accept positional arguments")
		return 2
	}
	if output == "" {
		fmt.Fprintln(stderr, "envsetter: pull requires --output")
		return 2
	}
	return pull(stdout, stderr, options, output, force)
}

func pull(stdout, stderr io.Writer, options commonOptions, output string, force bool) int {
	clientOptions, err := options.clientOptions()
	if err != nil {
		fmt.Fprintf(stderr, "envsetter: %v\n", err)
		return 1
	}
	plain, err := envvaultclient.Fetch(context.Background(), clientOptions)
	if err != nil {
		fmt.Fprintf(stderr, "envsetter: %v\n", err)
		return 1
	}
	if err := writeSecretFile(output, plain, force); err != nil {
		fmt.Fprintf(stderr, "envsetter: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", output)
	return 0
}

func runCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var options commonOptions
	addCommonFlags(fs, &options)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	command := fs.Args()
	if len(command) == 0 {
		fmt.Fprintln(stderr, "envsetter: run requires a command after --")
		return 2
	}

	clientOptions, err := options.clientOptions()
	if err != nil {
		fmt.Fprintf(stderr, "envsetter: %v\n", err)
		return 1
	}
	vars, err := envvaultclient.Load(context.Background(), clientOptions)
	if err != nil {
		fmt.Fprintf(stderr, "envsetter: %v\n", err)
		return 1
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = commandEnvironment(os.Environ(), vars)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "envsetter: starting command: %v\n", err)
		return 1
	}
	return 0
}

func (o commonOptions) clientOptions() (envvaultclient.Options, error) {
	token, err := credential(o.tokenFile, "vault token")
	if err != nil {
		return envvaultclient.Options{}, err
	}
	key, err := credential(o.keyFile, "decryption key")
	if err != nil {
		return envvaultclient.Options{}, err
	}
	clientOptions := envvaultclient.FromEnv(envvaultclient.Options{
		URL:     o.url,
		Token:   token,
		Key:     key,
		Vault:   normalizeVault(o.vault),
		Secret:  o.secret,
		Timeout: 15 * time.Second,
	})
	if transport != nil {
		clientOptions.HTTPClient = &http.Client{Transport: transport}
	}
	return clientOptions, nil
}

func credential(path, label string) (string, error) {
	if path == "" {
		return "", nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("read %s file: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s file %s must be a regular file", label, path)
	}
	if info.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("%s file %s must not be accessible by group or other users", label, path)
	}
	value, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s file: %w", label, err)
	}
	return strings.TrimSpace(string(value)), nil
}

func normalizeVault(vault string) string {
	return strings.TrimPrefix(vault, "envvault:")
}

func writeSecretFile(path string, contents []byte, force bool) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("inspect destination directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New("destination parent is not a directory")
	}
	if info.Mode().Perm()&0002 != 0 {
		return errors.New("destination directory is writable by other users")
	}

	if !force {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return errors.New("destination already exists; use --force to replace it")
			}
			return fmt.Errorf("create destination: %w", err)
		}
		return writeAndClose(file, contents)
	}

	temp, err := os.CreateTemp(dir, ".envsetter-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("secure temporary file: %w", err)
	}
	if err := writeAndClose(temp, contents); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace destination: %w", err)
	}
	return nil
}

func writeAndClose(file *os.File, contents []byte) error {
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return fmt.Errorf("write destination: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync destination: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close destination: %w", err)
	}
	return nil
}

func commandEnvironment(current []string, vars map[string]string) []string {
	values := make(map[string]string, len(current)+len(vars))
	for _, item := range current {
		name, value, ok := strings.Cut(item, "=")
		if ok && name != "ENVVAULT_TOKEN" && name != "ENVVAULT_KEY" {
			values[name] = value
		}
	}
	for name, value := range vars {
		values[name] = value
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	env := make([]string, 0, len(names))
	for _, name := range names {
		env = append(env, name+"="+values[name])
	}
	return env
}
