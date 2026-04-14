package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	defaultHost = "100.123.80.48"
	defaultUser = "seoy"
	defaultPort = "22"
)

type client struct {
	conn *ssh.Client
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
	if cfg.Password == "" {
		log.Fatal("set REMOTE_PASS")
	}

	c, err := dial(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	vmName := getenv("PODBRIDGE5_VM_NAME", "podbridge5-dev")
	vmRepo := getenv("PODBRIDGE5_VM_REPO", "/home/ubuntu/work/src/github.com/HeaInSeo/podbridge5")
	localRepo := getenv("PODBRIDGE5_LOCAL_REPO", "/opt/go/src/github.com/HeaInSeo/podbridge5")
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
			"if ! command -v buildah >/dev/null 2>&1 || ! command -v fuse-overlayfs >/dev/null 2>&1 || ! command -v pkg-config >/dev/null 2>&1 || ! pkg-config --exists gpgme >/dev/null 2>&1 || [ ! -f /usr/include/btrfs/version.h ] || ! command -v git >/dev/null 2>&1 || ! command -v go >/dev/null 2>&1 || ! command -v podman >/dev/null 2>&1; then sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y buildah fuse-overlayfs pkg-config libgpgme-dev libbtrfs-dev git golang-go podman; fi",
			fmt.Sprintf("mkdir -p %s", shellQuote(dirOf(vmRepo))),
			"sudo systemctl enable --now podman.socket",
			"sudo test -S /run/podman/podman.sock",
			"sudo podman info >/dev/null",
			"pkg-config --modversion gpgme",
		}
		fmt.Println(mustExec(c, vmName, strings.Join(commands, "; ")))
	case "sync":
		if err := syncWorktree(cfg, c, vmName, localRepo, vmRepo); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("synced %s -> %s on %s\n", localRepo, vmRepo, vmName)
	case "run":
		fmt.Println(mustExec(c, vmName, fmt.Sprintf("sudo bash -lc 'set -euo pipefail; cd %s; export XDG_RUNTIME_DIR=/run; export CONTAINER_HOST=unix:///run/podman/podman.sock; go test ./...'", shellQuote(vmRepo))))
	case "run-integration":
		fmt.Println(mustExec(c, vmName, fmt.Sprintf("sudo bash -lc 'set -euo pipefail; cd %s; export XDG_RUNTIME_DIR=/run; export CONTAINER_HOST=unix:///run/podman/podman.sock; unshare -r -m go test -v -tags=integration ./...'", shellQuote(vmRepo))))
	case "delete":
		run(c, fmt.Sprintf("multipass delete -p %s >/dev/null 2>&1 || true", vmName))
		run(c, "multipass purge >/dev/null 2>&1 || true")
		fmt.Println("deleted", vmName)
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func syncWorktree(cfg config, c *client, vmName, localRepo, vmRepo string) error {
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
		fmt.Sprintf("sudo rm -rf %s", shellQuote(vmRepo)),
		fmt.Sprintf("sudo mkdir -p %s", shellQuote(dirOf(vmRepo))),
		fmt.Sprintf("sudo tar -xzf %s -C %s", shellQuote(vmArchive), shellQuote(dirOf(vmRepo))),
		fmt.Sprintf("sudo chown -R ubuntu:ubuntu %s", shellQuote(dirOf(vmRepo))),
		fmt.Sprintf("rm -f %s", shellQuote(vmArchive)),
		fmt.Sprintf("test -f %s/go.mod", shellQuote(vmRepo)),
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
	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.Password(cfg.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // lab-only machine
		Timeout:         30 * time.Second,
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

func dial(cfg config) (*client, error) {
	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.Password(cfg.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // lab-only machine
		Timeout:         30 * time.Second,
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", cfg.Host, err)
	}
	return &client{conn: conn}, nil
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
	return c.Run(fmt.Sprintf("multipass exec %s -- bash -lc %q", vmName, cmd))
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

func run(c *client, cmd string) {
	if _, err := c.Run(cmd); err != nil {
		log.Fatal(err)
	}
}

func mustRun(c *client, cmd string) string {
	out, err := c.Run(cmd)
	if err != nil {
		log.Fatal(err)
	}
	return out
}

func mustExec(c *client, vmName, script string) string {
	out, err := c.MultipassExec(vmName, script)
	if err != nil {
		log.Fatal(err)
	}
	return out
}
