package podbridge5

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// buildWorkerJobFileEnv names the environment variable used to signal and
// locate a pending single-build job for a re-exec'd worker process. Its
// presence (non-empty) is what RunBuildWorkerIfRequested checks.
const buildWorkerJobFileEnv = "PODBRIDGE5_BUILD_JOB_FILE"

// buildWorkerResultFD is the file descriptor number the worker writes its
// JSON buildWorkerResult to. Kept separate from stdout/stderr so build
// progress log output (which the parent tees through for operator
// visibility) never gets mixed with the structured result.
const buildWorkerResultFD = 3

// buildWorkerJob is the JSON-serializable job handed to a worker process via
// a temp file. cfg.BuildLog (io.Writer) is never populated here — see
// BuildDockerfileContentUserNamespaceCancelable's doc comment.
type buildWorkerJob struct {
	DockerfileContent string                   `json:"dockerfile_content"`
	Config            UserNamespaceBuildConfig `json:"config"`
}

// buildWorkerResult is the JSON result a worker process writes to fd
// buildWorkerResultFD before exiting.
type buildWorkerResult struct {
	ImageID string `json:"image_id"`
	Digest  string `json:"digest"`
	Err     string `json:"err,omitempty"`
}

// RunBuildWorkerIfRequested checks whether this process was spawned by
// BuildDockerfileContentUserNamespaceCancelable to perform exactly one
// build+push. If so, it performs that job, writes the JSON result to fd
// buildWorkerResultFD, and returns true — the caller must os.Exit(0)
// immediately (mirroring ReexecIfNeeded's existing contract): this process
// has no further useful work to do, and must not continue into normal
// server startup (it would then try to bind the same port as its parent).
// Returns false when the process was started normally.
//
// Callers must invoke this at the very top of main(), after any host-mode
// ReexecIfNeeded call. This function follows the job's UserNamespaceMode:
// external user namespace jobs inherit the namespace their runtime already
// supplied, while auto/reexec jobs preserve the historical rootless reexec
// behavior.
func RunBuildWorkerIfRequested() bool {
	jobFile := os.Getenv(buildWorkerJobFileEnv)
	if jobFile == "" {
		return false
	}

	mode, err := readBuildWorkerUserNamespaceMode(jobFile)
	if err != nil {
		writeBuildWorkerResult(buildWorkerResult{Err: err.Error()})
		return true
	}
	if reexecIfNeededForUserNamespaceMode(mode) {
		// The further reexec'd copy inherits buildWorkerJobFileEnv and will
		// re-enter this same check; this process itself is done.
		os.Exit(0)
	}

	result := runBuildWorkerJob(jobFile)
	writeBuildWorkerResult(result)
	return true
}

func readBuildWorkerUserNamespaceMode(jobFile string) (UserNamespaceMode, error) {
	//nolint:gosec // G304: jobFile is a temp path this package created (writeBuildWorkerJobFile), not external input.
	data, err := os.ReadFile(jobFile)
	if err != nil {
		return UserNamespaceModeAuto, fmt.Errorf("read job file: %w", err)
	}
	var job buildWorkerJob
	if unmarshalErr := json.Unmarshal(data, &job); unmarshalErr != nil {
		return UserNamespaceModeAuto, fmt.Errorf("decode job file: %w", unmarshalErr)
	}
	if err := validateUserNamespaceMode(job.Config.UserNamespaceMode); err != nil {
		return UserNamespaceModeAuto, err
	}
	return job.Config.UserNamespaceMode, nil
}

func runBuildWorkerJob(jobFile string) buildWorkerResult {
	//nolint:gosec // G304: jobFile is a temp path this package created (writeBuildWorkerJobFile), not external input.
	data, err := os.ReadFile(jobFile)
	if err != nil {
		return buildWorkerResult{Err: fmt.Sprintf("read job file: %v", err)}
	}
	var job buildWorkerJob
	if unmarshalErr := json.Unmarshal(data, &job); unmarshalErr != nil {
		return buildWorkerResult{Err: fmt.Sprintf("decode job file: %v", unmarshalErr)}
	}

	imageID, digest, err := BuildAndPushUserNamespace(context.Background(), job.Config, job.DockerfileContent)
	if err != nil {
		return buildWorkerResult{Err: err.Error()}
	}
	return buildWorkerResult{ImageID: imageID, Digest: digest}
}

func writeBuildWorkerResult(result buildWorkerResult) {
	data, err := json.Marshal(result)
	if err != nil {
		// Nothing further to report through — fall back to stderr so the
		// parent's failure mode is at least a visible log line rather than
		// a silently empty/invalid result on fd buildWorkerResultFD.
		fmt.Fprintf(os.Stderr, "podbridge5: marshal build worker result: %v\n", err)
		return
	}
	f := os.NewFile(uintptr(buildWorkerResultFD), "build-worker-result")
	if f == nil {
		fmt.Fprintln(os.Stderr, "podbridge5: fd", buildWorkerResultFD, "(build worker result) not available")
		return
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "podbridge5: write build worker result: %v\n", err)
	}
}

// BuildDockerfileContentUserNamespaceCancelable is like
// BuildAndPushUserNamespace but runs the build+push in a dedicated child
// process (a re-exec of the current binary), killing the whole child
// process group if ctx is canceled before the build finishes.
//
// This is the only reliable way to stop a build once it has started:
// neither this package's in-process build path nor the vendored
// go.podman.io/buildah executor checks ctx.Done() while a RUN instruction
// is executing (buildah/chroot.RunUsingChroot, used for the chroot
// isolation this package defaults to, doesn't even accept a
// context.Context) — canceling ctx on the in-process path lets the current
// instruction run to completion regardless. A child OS process, in
// contrast, can always be killed from outside without its cooperation.
//
// dockerfileContent and cfg are passed to the child via a JSON temp file.
// cfg.BuildLog is NOT sent across the process boundary (io.Writer isn't
// JSON-marshalable and wouldn't be meaningful in another process anyway) —
// instead, wire stdout/stderr directly: the child's Buildah build progress
// output goes to its own stdout/stderr, which this function pipes to the
// stdout/stderr parameters exactly as cfg.BuildLog would for the in-process
// path (a nil cfg.BuildLog already makes Buildah default to its own
// process's stdout, so the child needs no special-casing here).
func BuildDockerfileContentUserNamespaceCancelable(
	ctx context.Context, dockerfileContent string, cfg UserNamespaceBuildConfig, stdout, stderr io.Writer,
) (imageID, digestStr string, err error) {
	jobFile, cleanup, err := writeBuildWorkerJobFile(dockerfileContent, cfg)
	if err != nil {
		return "", "", err
	}
	defer cleanup()

	selfExe, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("resolve self executable: %w", err)
	}

	resultR, resultW, err := os.Pipe()
	if err != nil {
		return "", "", fmt.Errorf("create result pipe: %w", err)
	}
	defer resultR.Close()

	//nolint:gosec // selfExe is os.Executable() (this binary), not user input.
	cmd := exec.Command(selfExe)
	cmd.Env = append(os.Environ(), buildWorkerJobFileEnv+"="+jobFile)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.ExtraFiles = []*os.File{resultW}
	// Setpgid makes this child the leader of a new process group, so
	// killing -pid below reaches every descendant it spawns (including any
	// chroot-isolation RUN subprocess buildah starts), not just the direct
	// child.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if startErr := cmd.Start(); startErr != nil {
		_ = resultW.Close()
		return "", "", fmt.Errorf("start build worker: %w", startErr)
	}
	_ = resultW.Close() // parent's copy; the child keeps its own fd buildWorkerResultFD

	waitErrCh := make(chan error, 1)
	go func() { waitErrCh <- cmd.Wait() }()

	select {
	case <-ctx.Done():
		if killErr := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); killErr != nil {
			Log.Errorf("podbridge5: failed to kill build worker process group %d: %v", cmd.Process.Pid, killErr)
		}
		<-waitErrCh // reap; the exit status itself doesn't matter here
		return "", "", fmt.Errorf("build worker canceled: %w", ctx.Err())
	case waitErr := <-waitErrCh:
		return parseBuildWorkerResult(resultR, waitErr)
	}
}

func parseBuildWorkerResult(resultR *os.File, waitErr error) (imageID, digestStr string, err error) {
	resultBytes, readErr := io.ReadAll(resultR)
	if readErr != nil {
		return "", "", fmt.Errorf("read build worker result: %w", readErr)
	}
	var result buildWorkerResult
	if jsonErr := json.Unmarshal(resultBytes, &result); jsonErr != nil {
		if waitErr != nil {
			return "", "", fmt.Errorf("build worker exited: %w", waitErr)
		}
		return "", "", fmt.Errorf("decode build worker result: %w (raw: %q)", jsonErr, resultBytes)
	}
	if result.Err != "" {
		return "", "", errors.New(result.Err)
	}
	return result.ImageID, result.Digest, nil
}

func writeBuildWorkerJobFile(dockerfileContent string, cfg UserNamespaceBuildConfig) (path string, cleanup func(), err error) {
	cfg.BuildLog = nil // io.Writer isn't JSON-marshalable; see doc comment above.
	job := buildWorkerJob{DockerfileContent: dockerfileContent, Config: cfg}
	data, err := json.Marshal(job)
	if err != nil {
		return "", nil, fmt.Errorf("marshal build worker job: %w", err)
	}
	f, err := os.CreateTemp("", "podbridge5-build-job-*.json")
	if err != nil {
		return "", nil, fmt.Errorf("create build worker job file: %w", err)
	}
	cleanupFn := func() { _ = os.Remove(f.Name()) }
	if _, writeErr := f.Write(data); writeErr != nil {
		_ = f.Close()
		cleanupFn()
		return "", nil, fmt.Errorf("write build worker job file: %w", writeErr)
	}
	if closeErr := f.Close(); closeErr != nil {
		cleanupFn()
		return "", nil, fmt.Errorf("close build worker job file: %w", closeErr)
	}
	return f.Name(), cleanupFn, nil
}
