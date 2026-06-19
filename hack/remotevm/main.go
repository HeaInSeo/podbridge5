package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	defaultHost = "100.123.80.48"
	defaultUser = "seoy"
	defaultPort = "22"
)

type client struct {
	conn *ssh.Client
}

type shellClient struct {
	cfg config
}

type remoteClient interface {
	Close() error
	Run(cmd string) (string, error)
	MultipassExec(vmName, cmd string) (string, error)
}

type config struct {
	Host     string
	Port     string
	User     string
	Password string
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <create|prepare|sync|run|run-integration|delete>", os.Args[0])
	}

	cfg := defaultConfig()

	c, err := dial(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	vmName := getenv("PODBRIDGE5_VM_NAME", "podbridge5-dev")
	vmRepo := getenv("PODBRIDGE5_VM_REPO", "/home/ubuntu/work/src/github.com/HeaInSeo/podbridge5")
	localRepo := getenv("PODBRIDGE5_LOCAL_REPO", "/opt/go/src/github.com/HeaInSeo/podbridge5")
	goVersion := getenv("PODBRIDGE5_GO_VERSION", "1.25.6")
	cpus := getenv("PODBRIDGE5_VM_CPUS", "2")
	memory := getenv("PODBRIDGE5_VM_MEMORY", "4G")
	disk := getenv("PODBRIDGE5_VM_DISK", "20G")

	switch os.Args[1] {
	case "create":
		run(c, fmt.Sprintf("multipass delete -p %s >/dev/null 2>&1 || true", vmName))
		run(c, "multipass purge >/dev/null 2>&1 || true")
		run(c, fmt.Sprintf("multipass launch 24.04 --name %s --cpus %s --memory %s --disk %s", vmName, cpus, memory, disk))
		fmt.Println(mustRun(c, fmt.Sprintf("multipass info %s", vmName)))
	case "prepare":
		commands := []string{
			"set -euo pipefail",
			"current_go_version() { if command -v go >/dev/null 2>&1; then go env GOVERSION 2>/dev/null || go version | cut -d\" \" -f3; else echo missing; fi; }",
			"if ! command -v buildah >/dev/null 2>&1 || ! command -v fuse-overlayfs >/dev/null 2>&1 || ! command -v pkg-config >/dev/null 2>&1 || ! pkg-config --exists gpgme >/dev/null 2>&1 || [ ! -f /usr/include/btrfs/version.h ] || ! command -v git >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1 || ! command -v tar >/dev/null 2>&1 || ! command -v podman >/dev/null 2>&1 || ! command -v gcc >/dev/null 2>&1; then sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y buildah fuse-overlayfs pkg-config libgpgme-dev libbtrfs-dev git curl tar podman gcc g++; fi",
			fmt.Sprintf("if [ \"$(current_go_version)\" != %q ]; then curl -fsSL %q -o /tmp/podbridge5-go.tar.gz && sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/podbridge5-go.tar.gz && rm -f /tmp/podbridge5-go.tar.gz; fi", "go"+goVersion, fmt.Sprintf("https://go.dev/dl/go%s.linux-amd64.tar.gz", goVersion)),
			"export PATH=/usr/local/go/bin:$PATH",
			fmt.Sprintf("[ \"$(current_go_version)\" = %q ]", "go"+goVersion),
			"if ! grep -q \"^ubuntu:\" /etc/subuid; then echo \"ubuntu:100000:65536\" | sudo tee -a /etc/subuid >/dev/null; fi",
			"if ! grep -q \"^ubuntu:\" /etc/subgid; then echo \"ubuntu:100000:65536\" | sudo tee -a /etc/subgid >/dev/null; fi",
			"if [ -e /proc/sys/kernel/apparmor_restrict_unprivileged_userns ]; then sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0 >/dev/null; fi",
			fmt.Sprintf("mkdir -p %q", dirOf(vmRepo)),
			"sudo systemctl enable --now podman.socket",
			"sudo test -S /run/podman/podman.sock",
			"sudo podman info >/dev/null",
			// run-integration connects as the unprivileged ubuntu user (it
			// needs unshare -r -m for unprivileged mount/overlay tests), so
			// it needs its own rootless podman socket, not the root-owned
			// system one. enable-linger keeps that user service running
			// outside an interactive login session.
			"sudo loginctl enable-linger ubuntu",
			"systemctl --user enable --now podman.socket",
			"test -S /run/user/1000/podman/podman.sock",
			"podman info >/dev/null",
			"pkg-config --modversion gpgme",
			"go version",
		}
		fmt.Println(mustExec(c, vmName, strings.Join(commands, "; ")))
	case "sync":
		if err := syncWorktree(cfg, c, vmName, localRepo, vmRepo); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("synced %s -> %s on %s\n", localRepo, vmRepo, vmName)
	case "run":
		testCmd := "sudo env PATH=/usr/local/go/bin:$PATH CGO_ENABLED=1 XDG_RUNTIME_DIR=/run CONTAINER_HOST=unix:///run/podman/podman.sock go test -v -tags=runtime -coverprofile=coverage-runtime.out -coverpkg=./... ./... ; sudo chown $(id -u):$(id -g) coverage-runtime.out"
		runTestsAndFetch(cfg, c, vmName, vmRepo, localRepo, testCmd, "test-runtime.log", "coverage-runtime.out")
	case "run-integration":
		testCmd := "env PATH=/usr/local/go/bin:$PATH CGO_ENABLED=1 XDG_RUNTIME_DIR=/run/user/1000 CONTAINER_HOST=unix:///run/user/1000/podman/podman.sock unshare -r -m go test -v -tags=runtime,integration -coverprofile=coverage-runtime-integration.out -coverpkg=./... ./..."
		runTestsAndFetch(cfg, c, vmName, vmRepo, localRepo, testCmd, "test-runtime-integration.log", "coverage-runtime-integration.out")
	case "delete":
		run(c, fmt.Sprintf("multipass delete -p %s >/dev/null 2>&1 || true", vmName))
		run(c, "multipass purge >/dev/null 2>&1 || true")
		fmt.Println("deleted", vmName)
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func syncWorktree(cfg config, c remoteClient, vmName, localRepo, vmRepo string) error {
	archivePath, cleanup, err := archiveWorktree(localRepo, filepath.Base(vmRepo))
	if err != nil {
		return err
	}
	defer cleanup()

	remoteArchive := fmt.Sprintf("/home/%s/%s-worktree.tar.gz", cfg.User, vmName)
	vmArchive := fmt.Sprintf("/home/ubuntu/%s-worktree.tar.gz", vmName)

	if _, err := c.Run(fmt.Sprintf("rm -f %s", shellQuote(remoteArchive))); err != nil {
		return fmt.Errorf("remove stale remote archive: %w", err)
	}
	if err := uploadFile(cfg, archivePath, remoteArchive); err != nil {
		return err
	}
	defer run(c, fmt.Sprintf("rm -f %s >/dev/null 2>&1 || true", shellQuote(remoteArchive)))

	if _, err := c.Run(fmt.Sprintf("multipass transfer %s %s:%s", shellQuote(remoteArchive), vmName, shellQuote(vmArchive))); err != nil {
		return fmt.Errorf("multipass transfer to %s: %w", vmName, err)
	}

	commands := []string{
		"set -euo pipefail",
		fmt.Sprintf("sudo rm -rf %q", vmRepo),
		fmt.Sprintf("sudo mkdir -p %q", dirOf(vmRepo)),
		fmt.Sprintf("sudo tar -xzf %q -C %q", vmArchive, dirOf(vmRepo)),
		fmt.Sprintf("sudo chown -R ubuntu:ubuntu %q", dirOf(vmRepo)),
		fmt.Sprintf("rm -f %q", vmArchive),
		fmt.Sprintf("test -f %q", filepath.Join(vmRepo, "go.mod")),
	}
	if _, err := c.MultipassExec(vmName, strings.Join(commands, "; ")); err != nil {
		return fmt.Errorf("extract synced worktree on %s: %w", vmName, err)
	}
	return nil
}

func archiveWorktree(localRepo, archiveRoot string) (string, func(), error) {
	info, err := os.Stat(localRepo)
	if err != nil {
		return "", nil, fmt.Errorf("stat local repo %s: %w", localRepo, err)
	}
	if !info.IsDir() {
		return "", nil, fmt.Errorf("local repo is not a directory: %s", localRepo)
	}

	tmpDir, err := os.MkdirTemp("", "podbridge5-remotevm-")
	if err != nil {
		return "", nil, fmt.Errorf("make temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }

	archivePath := filepath.Join(tmpDir, "podbridge5-worktree.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("create archive: %w", err)
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()
	tr := tar.NewWriter(gz)
	defer tr.Close()

	excluded := map[string]struct{}{
		".git":      {},
		".idea":     {},
		"artifacts": {},
	}

	err = filepath.Walk(localRepo, func(path string, entry os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == localRepo {
			return nil
		}

		rel, err := filepath.Rel(localRepo, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		top := strings.Split(rel, "/")[0]
		if _, skip := excluded[top]; skip {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		linkTarget := ""
		if entry.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(path)
			if err != nil {
				return err
			}
		}

		header, err := tar.FileInfoHeader(entry, linkTarget)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(archiveRoot, rel))
		if entry.IsDir() && !strings.HasSuffix(header.Name, "/") {
			header.Name += "/"
		}
		if err := tr.WriteHeader(header); err != nil {
			return err
		}
		if !entry.Mode().IsRegular() {
			return nil
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		if _, err := io.Copy(tr, src); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("archive local repo %s: %w", localRepo, err)
	}
	if err := tr.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close tar writer: %w", err)
	}
	if err := gz.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close gzip writer: %w", err)
	}
	if err := file.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close archive file: %w", err)
	}

	return archivePath, cleanup, nil
}

func uploadFile(cfg config, localPath, remotePath string) error {
	if cfg.Password == "" {
		return uploadFileWithCLI(cfg, localPath, remotePath)
	}

	sshCfg, err := newSSHClientConfig(cfg)
	if err != nil {
		return err
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial for upload: %w", err)
	}
	defer conn.Close()

	sess, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("new upload session: %w", err)
	}
	defer sess.Close()

	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open archive %s: %w", localPath, err)
	}
	defer file.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return fmt.Errorf("open upload stdin: %w", err)
	}

	copyErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(stdin, file)
		_ = stdin.Close()
		copyErr <- err
	}()

	if err := sess.Run(fmt.Sprintf("cat > %s", shellQuote(remotePath))); err != nil {
		return fmt.Errorf("upload archive to %s: %w", remotePath, err)
	}
	if err := <-copyErr; err != nil {
		return fmt.Errorf("stream archive to remote host: %w", err)
	}
	return nil
}

func uploadFileWithCLI(cfg config, localPath, remotePath string) error {
	target := fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath)
	args := []string{"-o", "BatchMode=yes", "-P", cfg.Port, localPath, target}
	cmd := exec.Command("scp", args...)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("scp upload to %s: %w\n%s", target, err, result)
	}
	return nil
}

// runTestsAndFetch runs testCmd inside the VM with its output redirected to
// a log file rather than streamed back through `multipass exec` directly.
// Streaming very verbose output (go test -v plus a cold `go mod download`)
// through multipass exec's exec channel has been observed to break the
// connection partway through with no error detail; redirecting to a file
// and pulling it back via `multipass transfer` (the same mechanism used for
// the worktree archive and coverage profiles) avoids that entirely.
func runTestsAndFetch(cfg config, c remoteClient, vmName, vmRepo, localRepo, testCmd, logName, coverageName string) {
	script := strings.Join([]string{
		fmt.Sprintf("cd %q || exit 1", vmRepo),
		fmt.Sprintf("rm -f %q", logName),
		fmt.Sprintf("{ %s ; } > %q 2>&1", testCmd, logName),
		fmt.Sprintf("echo \"EXITCODE=$?\" >> %q", logName),
	}, "; ")
	if _, err := c.MultipassExec(vmName, script); err != nil {
		log.Fatalf("run %s on %s: %v", logName, vmName, err)
	}

	if err := fetchRemoteFile(cfg, c, vmName, vmRepo, localRepo, logName); err != nil {
		log.Fatal(err)
	}
	logPath := filepath.Join(localRepo, "artifacts", logName)
	data, err := os.ReadFile(logPath)
	if err != nil {
		log.Fatalf("read fetched log %s: %v", logPath, err)
	}
	fmt.Print(string(data))

	if err := fetchRemoteFile(cfg, c, vmName, vmRepo, localRepo, coverageName); err != nil {
		log.Fatal(err)
	}

	if exitCode := lastExitCode(string(data)); exitCode != "0" {
		log.Fatalf("%s failed on %s (exit code %s) - see %s", logName, vmName, exitCode, logPath)
	}
}

func lastExitCode(logContent string) string {
	lines := strings.Split(strings.TrimRight(logContent, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if rest, ok := strings.CutPrefix(lines[i], "EXITCODE="); ok {
			return strings.TrimSpace(rest)
		}
	}
	return "unknown"
}

// fetchRemoteFile pulls a file written inside the VM (a -coverprofile or a
// redirected test log) back to localRepo/artifacts via `multipass transfer`
// followed by scp/ssh, so it can be inspected or merged locally.
func fetchRemoteFile(cfg config, c remoteClient, vmName, vmRepo, localRepo, fileName string) error {
	remoteHostPath := fmt.Sprintf("/home/%s/%s-%s", cfg.User, vmName, fileName)
	if _, err := c.Run(fmt.Sprintf("rm -f %s", shellQuote(remoteHostPath))); err != nil {
		return fmt.Errorf("remove stale remote file: %w", err)
	}
	defer run(c, fmt.Sprintf("rm -f %s >/dev/null 2>&1 || true", shellQuote(remoteHostPath)))

	vmPath := filepath.Join(vmRepo, fileName)
	if _, err := c.Run(fmt.Sprintf("multipass transfer %s:%s %s", vmName, shellQuote(vmPath), shellQuote(remoteHostPath))); err != nil {
		return fmt.Errorf("multipass transfer %s from %s: %w", fileName, vmName, err)
	}

	artifactsDir := filepath.Join(localRepo, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0o755); err != nil {
		return fmt.Errorf("create artifacts dir: %w", err)
	}
	localPath := filepath.Join(artifactsDir, fileName)
	if err := downloadFile(cfg, remoteHostPath, localPath); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "fetched %s -> %s\n", fileName, localPath)
	return nil
}

func downloadFile(cfg config, remotePath, localPath string) error {
	if cfg.Password == "" {
		return downloadFileWithCLI(cfg, remotePath, localPath)
	}

	sshCfg, err := newSSHClientConfig(cfg)
	if err != nil {
		return err
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		return fmt.Errorf("ssh dial for download: %w", err)
	}
	defer conn.Close()

	sess, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("new download session: %w", err)
	}
	defer sess.Close()

	outFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file %s: %w", localPath, err)
	}
	defer outFile.Close()

	sess.Stdout = outFile
	if err := sess.Run(fmt.Sprintf("cat %s", shellQuote(remotePath))); err != nil {
		return fmt.Errorf("download %s: %w", remotePath, err)
	}
	return nil
}

func downloadFileWithCLI(cfg config, remotePath, localPath string) error {
	source := fmt.Sprintf("%s@%s:%s", cfg.User, cfg.Host, remotePath)
	args := []string{"-o", "BatchMode=yes", "-P", cfg.Port, source, localPath}
	cmd := exec.Command("scp", args...)
	out, err := cmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("scp download from %s: %w\n%s", source, err, result)
	}
	return nil
}

func defaultConfig() config {
	host := getenv("REMOTE_HOST", defaultHost)
	user := getenv("REMOTE_USER", defaultUser)
	port := getenv("REMOTE_PORT", defaultPort)
	return config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: os.Getenv("REMOTE_PASS"),
	}
}

func dial(cfg config) (remoteClient, error) {
	if cfg.Password == "" {
		return &shellClient{cfg: cfg}, nil
	}

	sshCfg, err := newSSHClientConfig(cfg)
	if err != nil {
		return nil, err
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", cfg.Host, err)
	}
	return &client{conn: conn}, nil
}

func newSSHClientConfig(cfg config) (*ssh.ClientConfig, error) {
	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // lab-only machine
		Timeout:         30 * time.Second,
	}, nil
}

func authMethods(cfg config) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if cfg.Password != "" {
		methods = append(methods, ssh.Password(cfg.Password))
	}

	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		conn, err := net.Dial("unix", sock)
		if err == nil {
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}

	for _, candidate := range []string{"id_ed25519", "id_rsa"} {
		signer, err := loadSigner(candidate)
		if err == nil {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}

	if len(methods) == 0 {
		return nil, errors.New("configure REMOTE_PASS, SSH_AUTH_SOCK, or a default SSH private key")
	}
	return methods, nil
}

func loadSigner(name string) (ssh.Signer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(home, ".ssh", name)
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(keyData)
}

func (c *client) Close() error {
	return c.conn.Close()
}

func (c *client) Run(cmd string) (string, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	out, err := sess.CombinedOutput(cmd)
	result := strings.TrimSpace(string(out))
	if err != nil {
		return result, fmt.Errorf("cmd %q: %w\n%s", cmd, err, result)
	}
	return result, nil
}

func (c *client) MultipassExec(vmName, cmd string) (string, error) {
	return c.Run(fmt.Sprintf("multipass exec %s -- bash -lc %s", vmName, shellQuote(cmd)))
}

func (c *shellClient) Close() error {
	return nil
}

func (c *shellClient) Run(cmd string) (string, error) {
	target := fmt.Sprintf("%s@%s", c.cfg.User, c.cfg.Host)
	args := []string{"-o", "BatchMode=yes", "-p", c.cfg.Port, target, cmd}
	command := exec.Command("ssh", args...)
	out, err := command.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if err != nil {
		return result, fmt.Errorf("ssh cmd %q: %w\n%s", cmd, err, result)
	}
	return result, nil
}

func (c *shellClient) MultipassExec(vmName, cmd string) (string, error) {
	return c.Run(fmt.Sprintf("multipass exec %s -- bash -lc %s", vmName, shellQuote(cmd)))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func dirOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx <= 0 {
		return "."
	}
	return path[:idx]
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `"'"'`) + "'"
}

func run(c remoteClient, cmd string) {
	if _, err := c.Run(cmd); err != nil {
		log.Fatal(err)
	}
}

func mustRun(c remoteClient, cmd string) string {
	out, err := c.Run(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return out
}

func mustExec(c remoteClient, vmName, script string) string {
	out, err := c.MultipassExec(vmName, script)
	if err != nil {
		log.Fatal(err)
	}
	return out
}
