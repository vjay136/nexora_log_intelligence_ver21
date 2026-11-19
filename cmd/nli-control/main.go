package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"nexora.local/nli-v20/internal/auth"
	"nexora.local/nli-v20/internal/controlplane"
	"nexora.local/nli-v20/internal/engine"
	"nexora.local/nli-v20/internal/recovery"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
)

func loadConfig(path string) (controlplane.Config, error) {
	cfg := controlplane.Config{
		ListenAddress:             "127.0.0.1:12382",
		StateDir:                  "/var/lib/nli-controller",
		WebDir:                    "/usr/share/nli/webview_ver20",
		EnrollmentTokenEnv:        "NLI_ENROLLMENT_TOKEN",
		BootstrapAdminPasswordEnv: "NLI_BOOTSTRAP_ADMIN_PASSWORD",
		CommandLeaseSeconds:       120,
		EventRetention:            50000,
	}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("decode config %s: %w", path, err)
		}
		return cfg, nil
	}
	if !os.IsNotExist(err) {
		return cfg, err
	}
	fmt.Fprintf(os.Stderr, "nli-control: config file %s not found, using defaults\n", path)
	return cfg, nil
}

func resetAdminPassword(cfg controlplane.Config, input io.Reader) error {
	scanner := bufio.NewScanner(io.LimitReader(input, 4097))
	scanner.Buffer(make([]byte, 128), 4096)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return fmt.Errorf("new admin password is required on stdin")
	}
	password := strings.TrimSuffix(scanner.Text(), "\r")
	if scanner.Scan() {
		return fmt.Errorf("stdin must contain exactly one password line")
	}
	store, err := auth.NewStoreWithBootstrap(filepath.Join(cfg.StateDir, "users.json"), "")
	if err != nil {
		return fmt.Errorf("load auth store: %w", err)
	}
	if err := store.ResetPassword("admin", password); err != nil {
		return err
	}
	return nil
}

func restoreRecoveryCapsule(args []string) error {
	flags := flag.NewFlagSet("recovery-restore", flag.ContinueOnError)
	source := flags.String("capsule", "", "encrypted capsule downloaded from Disaster Recovery")
	keyFile := flags.String("passphrase-file", "", "protected file containing the capsule passphrase")
	state := flags.String("state-dir", "/var/lib/nli-controller", "controller state directory")
	config := flags.String("config-dir", "/etc/nli", "controller configuration directory")
	stopped := flags.Bool("confirm-stopped", false, "confirm that the controller service is stopped")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*stopped || *source == "" || *keyFile == "" {
		return fmt.Errorf("--capsule, --passphrase-file and --confirm-stopped are required")
	}
	if err := recoveryStopped(); err != nil {
		return err
	}
	ownerID, groupID := -1, -1
	if runtime.GOOS == "linux" {
		account, err := user.Lookup("nli")
		if err != nil {
			return fmt.Errorf("install the controller RPM before recovery: %w", err)
		}
		ownerID, err = strconv.Atoi(account.Uid)
		if err != nil {
			return err
		}
		groupID, err = strconv.Atoi(account.Gid)
		if err != nil {
			return err
		}
	}
	info, err := os.Stat(*keyFile)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("passphrase file must be mode 0600")
	}
	passphrase, err := os.ReadFile(*keyFile)
	if err != nil || len(passphrase) > 1026 {
		return fmt.Errorf("invalid recovery passphrase file")
	}
	encoded, err := recovery.ReadCapsule(*source)
	if err != nil {
		return err
	}
	contents, err := recovery.Open(encoded, strings.TrimSuffix(strings.TrimSuffix(string(passphrase), "\n"), "\r"))
	if err != nil {
		return err
	}
	if filepath.Clean(contents.StateDir) != filepath.Clean(*state) || filepath.Clean(contents.ConfigDir) != filepath.Clean(*config) {
		return fmt.Errorf("restore must use the original state and configuration roots (%s and %s); recover at those paths so stored TLS and metadata paths stay valid", contents.StateDir, contents.ConfigDir)
	}
	if err := recovery.RestoreOwned(contents, *state, *config, ownerID, groupID); err != nil {
		return err
	}
	fmt.Printf("Controller identity restored (%d files). Confirm endpoint reachability and Supabase credentials, then start nli-controller-agent.service.\n", len(contents.Files))
	return nil
}

func recoveryStopped() error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("controller recovery requires root")
	}
	err := exec.Command("systemctl", "is-active", "--quiet", "nli-controller-agent.service").Run()
	if err == nil {
		return fmt.Errorf("stop nli-controller-agent.service before controller recovery")
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || (exit.ExitCode() != 3 && exit.ExitCode() != 4) {
		return fmt.Errorf("unable to verify stopped controller service: %w", err)
	}
	return nil
}

func backupRecoveryCapsule(args []string) error {
	flags := flag.NewFlagSet("recovery-backup", flag.ContinueOnError)
	output := flags.String("output", "", "new encrypted capsule output file")
	keyFile := flags.String("passphrase-file", "", "protected recovery passphrase file")
	state := flags.String("state-dir", "/var/lib/nli-controller", "controller state directory")
	config := flags.String("config-dir", "/etc/nli", "controller configuration directory")
	stopped := flags.Bool("confirm-stopped", false, "confirm the controller service is stopped for a consistent snapshot")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*stopped || *output == "" || *keyFile == "" {
		return fmt.Errorf("--output, --passphrase-file and --confirm-stopped are required")
	}
	if err := recoveryStopped(); err != nil {
		return err
	}
	info, err := os.Stat(*keyFile)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("passphrase file must be mode 0600")
	}
	passphrase, err := os.ReadFile(*keyFile)
	if err != nil || len(passphrase) > 1026 {
		return fmt.Errorf("invalid recovery passphrase file")
	}
	contents, err := recovery.Capture(*state, *config)
	if err != nil {
		return err
	}
	encoded, err := recovery.Seal(contents, strings.TrimSuffix(strings.TrimSuffix(string(passphrase), "\n"), "\r"))
	if err != nil {
		return err
	}
	if err = recovery.WriteCapsule(*output, encoded); err != nil {
		return err
	}
	fmt.Printf("Encrypted controller recovery capsule written (%d files). Keep the passphrase in a separate protected location. Restart the controller and import the encrypted file in Disaster Recovery.\n", len(contents.Files))
	return nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "recovery-backup" {
		if err := backupRecoveryCapsule(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "nli-control: recovery backup:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "recovery-restore" {
		if err := restoreRecoveryCapsule(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "nli-control: recovery restore:", err)
			os.Exit(1)
		}
		return
	}

	fs := flag.NewFlagSet("nli-control", flag.ExitOnError)
	configFile := fs.String("config", "/etc/nli/controller.json", "control plane config file")
	versionFlag := fs.Bool("version", false, "print version")
	resetAdmin := fs.Bool("reset-admin-password-stdin", false, "reset the existing admin password from one stdin line while the service is stopped")
	fs.Parse(os.Args[1:])

	if *versionFlag {
		fmt.Println("nli-control", engine.Version)
		return
	}

	cfg, err := loadConfig(*configFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nli-control:", err)
		os.Exit(1)
	}
	if *resetAdmin {
		if err := resetAdminPassword(cfg, os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, "nli-control: reset admin password:", err)
			os.Exit(1)
		}
		fmt.Println("nli-control: admin password reset")
		return
	}

	if err := os.MkdirAll(cfg.StateDir, 0750); err != nil {
		fmt.Fprintln(os.Stderr, "nli-control: state dir:", err)
		os.Exit(1)
	}

	srv, err := controlplane.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nli-control:", err)
		os.Exit(1)
	}

	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "nli-control:", err)
		os.Exit(1)
	}
}
