# Runtime 검증

이 문서는 `podbridge5`의 runtime 검증 경로를 설명합니다.
README 본문은 프로젝트 소개에 집중하고, VM 기반 테스트 절차는 여기로 분리합니다.

runtime 초기화 우선순위 자체는 [runtime-policy.ko.md](runtime-policy.ko.md)를 따릅니다.
즉, host preflight와 실제 Go 코드 모두 `CONTAINER_HOST -> XDG_RUNTIME_DIR -> 기본 socket` 순서를 사용합니다.

이 문서의 모든 경로는 `Go 1.25.6` 기준을 전제로 합니다.
`podbridge5`는 다른 관련 프로젝트와 같은 `Go 1.25.x` baseline을 사용하며, 낮은 Go 버전 호환은 목표가 아닙니다.

## 검증 경로

검증은 세 층으로 나눕니다.

### 1. 로컬 unit 경로

현재 호스트에서 빠르게 확인하는 경로입니다.

- `make test-unit`
- `make runtime-env-check`
- `make test-runtime`
- `make test-runtime-integration`

이 타깃들은 실행 전에 `go-version-check`를 거쳐 현재 `go` 바이너리가 `go1.25.6`인지 확인합니다.

이 경로는 코드 수정과 경량 검증에 적합합니다.
`test-unit`은 runtime 태그 테스트를 제외한 빠른 경로이고, `test-runtime`은 현재 호스트에서 Podman/buildah 환경까지 포함해 확인하는 경로입니다.
다만 `buildah`, `containers/storage`, `containers/image` 의존성 때문에 runtime 경로는 호스트 패키지 준비가 필요할 수 있습니다.

### 2. 로컬 runtime / integration 경로

현재 호스트에서 실제 Podman/buildah 환경을 붙여 보는 경로입니다.

- `make test-runtime`
- `make test-runtime-integration`

`test-runtime`은 `runtime` 태그 테스트를 수행합니다. 실행 전 `runtime-host-check`가 현재 호스트의 Podman socket 상태를 확인합니다.
`test-runtime-integration`은 `runtime + integration` 태그 테스트를 `unshare` 환경에서 수행합니다. 실행 전 `runtime-integration-host-check`가 `unshare` 사용 가능 여부를 확인합니다.

### 3. 원격 clean VM 경로

현재는 옆 장비의 VM 작업이 진행 중이면 이 경로를 잠시 보류할 수 있습니다.
그 경우에는 `test-unit`과 로컬 preflight 중심으로 진행하고, VM 복구 후 `vm-test-runtime*` 경로를 재개하면 됩니다.

runtime 의존 검증은 `100.123.80.48` 장비의 Multipass에서 ephemeral VM을 만들어 수행합니다.

#### 수동 디버깅용 영구 VM

`make vm-test-runtime`은 ephemeral VM을 만들고 테스트 후 삭제한다.
rootless/rootful 동작을 직접 확인하거나 buildah/podman 문제를 파고들 때는 영구 VM이 필요하다.

```bash
# 영구 dev VM 생성 및 패키지 설치
bash hack/setup-dev-vm.sh

# 커스텀 설정
REMOTE_HOST=100.123.80.48 REMOTE_USER=seoy bash hack/setup-dev-vm.sh
```

VM이 이미 존재하면 생성을 건너뛴다 (idempotent).
설치 항목: `buildah`, `podman`, `fuse-overlayfs`, `slirp4netns`, `uidmap` + `/etc/subuid`, `/etc/subgid` 자동 구성.

VM을 삭제해도 이 스크립트로 동일한 환경을 재현할 수 있다.

- 대상 장비: `100.123.80.48`
- 사용자: `seoy`
- 기본 VM 이름: `podbridge5-dev`
- 권장 OS: Ubuntu 24.04
- 인증: SSH key/agent 우선, 필요 시 `REMOTE_PASS`

검증 대상 예시:

- `buildah`
- `fuse-overlayfs`
- `pkg-config` / `gpgme`
- `btrfs` headers
- system `podman` socket
- storage 초기화
- image build / push 흐름

## 왜 clean VM을 쓰는가

중요한 기준은 재현성입니다.
환경 찌꺼기 때문에 생기는 문제라면 VM을 지우고 다시 만들었을 때 사라져야 합니다.
반대로 fresh VM에서도 계속 재현되면, 그 문제는 코드나 runtime 초기화 경로에 있을 가능성이 큽니다.

## Makefile 자동화

원격 VM 테스트는 Makefile이 자동으로 처리합니다.

`make vm-test-runtime` 실행 순서:

1. 원격 장비에서 테스트용 VM 생성
2. 필요한 패키지와 system `podman` socket 준비
   이 단계에서 VM 내부 Go toolchain도 `1.25.6`으로 맞춥니다.
3. 현재 로컬 `podbridge5` 워크트리를 tar.gz로 묶어 원격 호스트로 업로드
4. 원격 호스트에서 `multipass transfer`로 fresh VM에 동기화
5. VM 안에서 `go test ./...` 실행
6. 테스트 종료 후 VM 삭제

`make vm-test-runtime-integration`도 같은 흐름으로 동작하며 `runtime + integration` 태그 테스트를 수행합니다.

## 로그 수집

원격 VM 테스트 출력은 콘솔에 표시되는 동시에 로컬 로그 파일로 저장됩니다.

- `artifacts/vm-test-runtime.log`
- `artifacts/vm-test-runtime-integration.log`

로그에는 VM lifecycle, 원격 준비 단계, worktree sync, `go test` stdout/stderr가 함께 들어갑니다.

로컬 runtime 경로가 실패할 때는 먼저 preflight 메시지를 보면 됩니다.

- `runtime-env-check`: buildah, fuse-overlayfs, gpgme, btrfs header 같은 패키지 문제
- `runtime-host-check`: `CONTAINER_HOST`, Podman socket, `XDG_RUNTIME_DIR` 문제
- `runtime-integration-host-check`: `unshare` 사용 가능 여부 문제

## 자주 쓰는 타깃

- `make test-unit`
- `make runtime-env-check`
- `make runtime-host-check`
- `make test-runtime`
- `make test-runtime-integration`
- `make vm-test-runtime REMOTE_PASS=...`
- `make vm-test-runtime-integration REMOTE_PASS=...`
- `make vm-create-runtime REMOTE_PASS=...`
- `make vm-prepare-runtime REMOTE_PASS=...`
- `make vm-sync-runtime REMOTE_PASS=...`
- `make vm-delete-runtime REMOTE_PASS=...`

기본값으로 현재 로컬 워크트리(`$(CURDIR)`)가 VM에 동기화됩니다.
필요하면 `PODBRIDGE5_LOCAL_REPO=/path/to/repo`로 명시적으로 바꿀 수 있습니다.

SSH key 또는 SSH agent가 이미 설정되어 있으면 `REMOTE_PASS` 없이도 실행할 수 있습니다.
현재 운영 기준에서는 `ssh seoy@100.123.80.48` 직접 접속이 되면 VM 경로를 우선 사용합니다.
