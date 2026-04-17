# Runtime 검증

이 문서는 `podbridge5`의 runtime 검증 경로를 설명합니다.
README 본문은 프로젝트 소개에 집중하고, VM 기반 테스트 절차는 여기로 분리합니다.

## 검증 경로

검증은 두 경로로 나눕니다.

### 1. 로컬 기본 경로

현재 호스트에서 빠르게 확인하는 경로입니다.

- `make test`
- `make test-runtime`
- `make test-runtime-integration`
- `make runtime-env-check`

이 경로는 코드 수정과 경량 검증에 적합합니다.
다만 `buildah`, `containers/storage`, `containers/image` 의존성 때문에 호스트 패키지가 없으면 전체 테스트가 바로 되지 않을 수 있습니다.

### 2. 원격 clean VM 경로

runtime 의존 검증은 `100.123.80.48` 장비의 Multipass에서 ephemeral VM을 만들어 수행합니다.

- 대상 장비: `100.123.80.48`
- 사용자: `seoy`
- 기본 VM 이름: `podbridge5-dev`
- 권장 OS: Ubuntu 24.04

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
3. 현재 로컬 `podbridge5` 워크트리를 tar.gz로 묶어 원격 호스트로 업로드
4. 원격 호스트에서 `multipass transfer`로 fresh VM에 동기화
5. VM 안에서 `go test ./...` 실행
6. 테스트 종료 후 VM 삭제

`make vm-test-runtime-integration`도 같은 흐름으로 동작하며 integration 태그 테스트를 수행합니다.

## 로그 수집

원격 VM 테스트 출력은 콘솔에 표시되는 동시에 로컬 로그 파일로 저장됩니다.

- `artifacts/vm-test-runtime.log`
- `artifacts/vm-test-runtime-integration.log`

로그에는 VM lifecycle, 원격 준비 단계, worktree sync, `go test` stdout/stderr가 함께 들어갑니다.

## 자주 쓰는 타깃

- `make vm-test-runtime REMOTE_PASS=...`
- `make vm-test-runtime-integration REMOTE_PASS=...`
- `make vm-create-runtime REMOTE_PASS=...`
- `make vm-prepare-runtime REMOTE_PASS=...`
- `make vm-sync-runtime REMOTE_PASS=...`
- `make vm-delete-runtime REMOTE_PASS=...`

기본값으로 현재 로컬 워크트리(`$(CURDIR)`)가 VM에 동기화됩니다.
필요하면 `PODBRIDGE5_LOCAL_REPO=/path/to/repo`로 명시적으로 바꿀 수 있습니다.
