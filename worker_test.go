//go:build !runtime

// This file's TestMain must not compile alongside runtime_main_test.go's
// (both declare TestMain — Go allows only one per package). These worker
// tests use a fake in-process stand-in worker (see testWorkerFakeModeEnv)
// specifically so they don't need the real buildah/containers-storage
// reexec machinery runtime_main_test.go's TestMain sets up, so excluding
// this file under the "runtime" tag (rather than merging the two TestMains)
// keeps both test suites independently meaningful.
package podbridge5

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// testWorkerFakeModeEnv, when set, makes this test binary's TestMain act as
// a stand-in worker instead of running go test's normal test suite — this
// lets BuildDockerfileContentUserNamespaceCancelable's process-management
// code (spawn, result-fd JSON framing, process-group kill on cancel) be
// exercised end-to-end without a real container storage backend. The
// buildah-specific logic inside runBuildWorkerJob (BuildAndPushUserNamespace)
// is covered separately by this package's existing runtime/integration
// tests, not re-tested here.
const testWorkerFakeModeEnv = "PODBRIDGE5_TEST_FAKE_WORKER"

func TestMain(m *testing.M) {
	if os.Getenv(testWorkerFakeModeEnv) != "" {
		runFakeWorkerForTest()
		return
	}
	os.Exit(m.Run())
}

func runFakeWorkerForTest() {
	jobFile := os.Getenv(buildWorkerJobFileEnv)
	data, err := os.ReadFile(jobFile)
	if err != nil {
		writeBuildWorkerResult(buildWorkerResult{Err: err.Error()})
		os.Exit(0)
	}
	var job buildWorkerJob
	if err := json.Unmarshal(data, &job); err != nil {
		writeBuildWorkerResult(buildWorkerResult{Err: err.Error()})
		os.Exit(0)
	}
	switch job.DockerfileContent {
	case "sleep-forever":
		select {} // blocks until killed by the parent — see the cancel test below
	case "fail":
		writeBuildWorkerResult(buildWorkerResult{Err: "simulated build failure"})
	default:
		writeBuildWorkerResult(buildWorkerResult{ImageID: "fake-image-id", Digest: "sha256:fakedigest"})
	}
	os.Exit(0)
}

func TestBuildDockerfileContentUserNamespaceCancelable_Success(t *testing.T) {
	t.Setenv(testWorkerFakeModeEnv, "1")
	var stdout, stderr bytes.Buffer
	imageID, digest, err := BuildDockerfileContentUserNamespaceCancelable(
		t.Context(), "FROM alpine", UserNamespaceBuildConfig{OutputRef: "example/test:latest"}, &stdout, &stderr,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, stderr.String())
	}
	if imageID != "fake-image-id" {
		t.Errorf("imageID = %q, want fake-image-id", imageID)
	}
	if digest != "sha256:fakedigest" {
		t.Errorf("digest = %q, want sha256:fakedigest", digest)
	}
}

func TestBuildDockerfileContentUserNamespaceCancelable_BuildFailure(t *testing.T) {
	t.Setenv(testWorkerFakeModeEnv, "1")
	_, _, err := BuildDockerfileContentUserNamespaceCancelable(
		t.Context(), "fail", UserNamespaceBuildConfig{OutputRef: "example/test:latest"}, io.Discard, io.Discard,
	)
	if err == nil || !strings.Contains(err.Error(), "simulated build failure") {
		t.Fatalf("err = %v, want it to contain 'simulated build failure'", err)
	}
}

// TestBuildDockerfileContentUserNamespaceCancelable_CancelKillsProcessGroup is
// the regression test for the whole point of this file: a build that would
// otherwise hang forever must actually stop when ctx is canceled. Before
// this package existed, canceling ctx on the in-process build path let the
// current instruction run to completion regardless — the equivalent
// scenario here (a worker process that blocks forever) must instead be
// killed and this call must return promptly.
func TestBuildDockerfileContentUserNamespaceCancelable_CancelKillsProcessGroup(t *testing.T) {
	t.Setenv(testWorkerFakeModeEnv, "1")
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, _, err := BuildDockerfileContentUserNamespaceCancelable(
			ctx, "sleep-forever", UserNamespaceBuildConfig{OutputRef: "example/test:latest"}, io.Discard, io.Discard,
		)
		done <- err
	}()

	// Give the child time to actually start blocking in select{} before we
	// cancel — canceling immediately would mostly test process-start
	// latency, not the kill path.
	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("BuildDockerfileContentUserNamespaceCancelable did not return after cancel — " +
			"process-group kill may have failed, leaving the worker running")
	}
}

func TestWriteBuildWorkerJobFile_RoundTripsConfigExcludingBuildLog(t *testing.T) {
	cfg := UserNamespaceBuildConfig{
		OutputRef:             "harbor.example.com/tool:1.0",
		ContextDirectory:      "/tmp/ctx",
		Runtime:               "crun",
		CacheRef:              "harbor.example.com/cache:layers",
		Isolation:             BuildIsolationChroot,
		UserNamespaceMode:     UserNamespaceModeExternal,
		StorageMode:           StorageNativeOverlay,
		InsecureSkipTLSVerify: true,
		BuildLog:              io.Discard, // must not survive the round trip
	}
	jobFile, cleanup, err := writeBuildWorkerJobFile("FROM alpine\nRUN true", cfg)
	t.Cleanup(cleanup)
	if err != nil {
		t.Fatalf("writeBuildWorkerJobFile: %v", err)
	}

	data, err := os.ReadFile(jobFile)
	if err != nil {
		t.Fatalf("read job file: %v", err)
	}
	var job buildWorkerJob
	if err := json.Unmarshal(data, &job); err != nil {
		t.Fatalf("decode job file: %v", err)
	}

	if job.DockerfileContent != "FROM alpine\nRUN true" {
		t.Errorf("DockerfileContent = %q", job.DockerfileContent)
	}
	if job.Config.OutputRef != cfg.OutputRef || job.Config.Runtime != cfg.Runtime ||
		job.Config.CacheRef != cfg.CacheRef || job.Config.Isolation != cfg.Isolation ||
		job.Config.UserNamespaceMode != cfg.UserNamespaceMode ||
		job.Config.StorageMode != cfg.StorageMode || job.Config.InsecureSkipTLSVerify != cfg.InsecureSkipTLSVerify {
		t.Errorf("config fields did not round-trip: got %+v", job.Config)
	}
	if job.Config.BuildLog != nil {
		t.Error("BuildLog must not survive serialization (io.Writer isn't JSON-marshalable)")
	}

	if _, statErr := os.Stat(jobFile); statErr != nil {
		t.Fatalf("job file should exist before cleanup: %v", statErr)
	}
	cleanup()
	if _, statErr := os.Stat(jobFile); !os.IsNotExist(statErr) {
		t.Errorf("job file should be removed after cleanup, stat err = %v", statErr)
	}
}

func TestShouldReexecForUserNamespaceMode(t *testing.T) {
	tests := []struct {
		name string
		mode UserNamespaceMode
		want bool
	}{
		{name: "auto preserves legacy reexec", mode: UserNamespaceModeAuto, want: true},
		{name: "explicit reexec", mode: UserNamespaceModeReexec, want: true},
		{name: "external runtime user namespace", mode: UserNamespaceModeExternal, want: false},
		{name: "unknown preserves legacy reexec", mode: "unknown", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldReexecForUserNamespaceMode(tc.mode); got != tc.want {
				t.Fatalf("shouldReexecForUserNamespaceMode(%q) = %v, want %v", tc.mode, got, tc.want)
			}
		})
	}
}
