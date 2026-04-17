# Sprint: Runtime Refactor And Test Split

English follows the Korean section.

## 목표

이번 스프린트의 목적은 `podbridge5`의 런타임 의존 코드와 순수 로직을 더 명확히 분리해서,
문제가 발생했을 때 원인을 빠르게 가르는 것입니다.

직전 스프린트에서 확보한 것은 다음입니다.

- fresh VM 기준 재현 가능한 runtime test
- 현재 로컬 worktree를 remote VM에 동기화하는 자동화
- VM lifecycle과 로그 수집 자동화

이제 필요한 것은 다음입니다.

- 런타임 계약의 명문화
- 순수 Go 검증 범위 확대
- runtime test의 실패 원인 분리

## 이번 스프린트 범위

### 1. Runtime contract 고정

우선 `executor.sh`, `healthcheck.sh`, `/app` 경로, 결과 로그 경로, 기본 command를 코드 상수로 고정합니다.
이 단계의 목적은 문자열이 여러 파일에 흩어져 있는 상태를 줄이고, 이후 refactor의 기준점을 만드는 것입니다.

대상:

- `container.go`
- `executor.go`
- `buildconfig.go`
- 관련 단위 테스트

완료 기준:

- `/app/executor.sh`, `/app/healthcheck.sh`, `/app/result.log`, `/app/exit_code.log`가 공통 상수로 관리됨
- healthcheck command 파싱이 별도 순수 함수로 분리됨
- 해당 로직이 unit test로 검증됨

상태:

- 완료

### 2. Image option assembly 분리

다음으로 `image.go`의 buildah 호출 전 단계에서 조립되는 값들을 순수 함수로 분리합니다.
이 단계의 목적은 build runtime 없이도 build option, push destination, builder option 기본값을 검증할 수 있게 만드는 것입니다.

대상:

- `image.go`
- 신규 pure helper
- 신규 unit test

완료 기준:

- build options 조립이 별도 순수 함수로 분리됨
- push destination 정규화가 별도 순수 함수로 분리됨
- builder/add-copy 기본 옵션이 별도 순수 함수로 분리됨
- 해당 로직이 unit test로 검증됨

상태:

- 이번 턴에서 구현 진행

### 3. Unit vs Runtime test 경계 정리

다음 단계에서는 테스트를 두 층으로 더 분명히 나눕니다.

- unit: 순수 로직, 파싱, 옵션 조립, 에러 규약
- runtime: podman/buildah/storage/socket에 실제로 붙는 검증

이번 스프린트에서는 우선 healthcheck/executor 계약과 image option assembly를 unit 친화적으로 정리하고,
그 다음 `volume` 쪽으로 같은 패턴을 확장합니다.

### 3. Volume mode decision 분리

다음으로 `volume.go`의 mode 판단을 runtime 호출 앞에서 순수 함수로 분리합니다.
이 단계의 목적은 `ModeSkip`, `ModeUpdate`, `ModeOverwrite`가 existing volume 상태와 결합될 때 어떤 동작을 선택하는지 build/runtime 없이 검증할 수 있게 만드는 것입니다.

대상:

- `volume.go`
- 신규 pure helper
- 신규 unit test

완료 기준:

- volume mode 결정이 별도 순수 함수로 분리됨
- `WriteFolderToVolume`가 그 결정 결과를 사용함
- 해당 로직이 unit test로 검증됨

상태:

- 이번 턴에서 구현 진행

### 4. Unit vs Runtime test 경계 정리

다음 단계에서는 테스트를 두 층으로 더 분명히 나눕니다.

- unit: 순수 로직, 파싱, 옵션 조립, 에러 규약
- runtime: podman/buildah/storage/socket에 실제로 붙는 검증

이번 스프린트에서는 우선 healthcheck/executor 계약, image option assembly, volume mode decision을 unit 친화적으로 정리하고,
그 다음 runtime-only 레이어 분리를 더 진행합니다.

### 4. Volume runtime helper 분리

다음으로 `volume.go` 안에 직접 섞여 있던 image/container runtime 호출을 별도 helper 레이어로 옮깁니다.
이 단계의 목적은 `WriteFolderToVolume`, `ReadDataFromVolume`가 정책 흐름과 tar 처리에 집중하고,
실제 Podman binding 호출은 더 얇은 경계 뒤로 밀어 넣는 것입니다.

대상:

- `volume.go`
- 신규 runtime helper
- 신규 unit test

완료 기준:

- volume writer/reader spec 조립이 dedicated helper로 분리됨
- image ensure, create/start, stop/remove가 runtime helper를 통해 호출됨
- `WriteFolderToVolume`와 `ReadDataFromVolume`가 runtime helper를 사용함
- helper 조립 로직이 unit test로 검증됨

상태:

- 이번 턴에서 구현 진행

### 5. Container runtime helper 분리

다음으로 `container.go` 안에 직접 섞여 있던 image/container runtime 호출을 별도 helper 레이어로 옮깁니다.
이 단계의 목적은 `CreateContainer`, `StartContainer`, `InspectContainer`가 흐름과 계약에 집중하고,
실제 Podman binding 호출과 상태 매핑은 더 얇은 경계 뒤로 밀어 넣는 것입니다.

대상:

- `container.go`
- 신규 runtime helper
- 신규 unit test

완료 기준:

- container exists / image ensure / create / start / inspect가 runtime helper를 통해 호출됨
- `CreateContainer`, `StartContainer`, `InspectContainer`, `handleExistingContainer`가 runtime helper를 사용함
- inspect state -> `ContainerStatus` 매핑이 별도 순수 함수로 분리됨
- helper와 상태 매핑 로직이 unit test로 검증됨

상태:

- 이번 턴에서 구현 진행

### 6. Image build/push runtime helper 분리

다음으로 `image.go`의 build/push 핵심 경로를 별도 runtime helper 레이어로 옮깁니다.
이 단계의 목적은 `BuildDockerfileContent`, `PushImage`, `BuildAndPushDockerfileContent`가 흐름과 입력 검증에 집중하고,
실제 buildah/imagebuildah 호출은 더 얇은 경계 뒤로 밀어 넣는 것입니다.

대상:

- `image.go`
- 신규 runtime helper
- 신규 unit test

완료 기준:

- build Dockerfile / push image 호출이 runtime helper를 통해 수행됨
- temp Dockerfile 작성과 build/push orchestration이 helper 함수로 분리됨
- `BuildDockerfileContent`, `PushImage`, `BuildAndPushDockerfileContent`가 runtime helper를 사용함
- helper orchestration 로직이 unit test로 검증됨

상태:

- 이번 턴에서 구현 진행

### 5. 다음 후속 작업 준비

이 스프린트가 끝나면 후속으로 바로 이어질 작업은 다음입니다.

- `image.go`와 `container.go`의 runtime-only 레이어 분리 심화
- dry-run / timeout 기능 구체화

## 이번 턴에서 바로 시작한 작업

이 문서를 기준으로 이번 턴까지 진행한 구현은 아래 여섯 가지입니다.

1. runtime contract 고정
2. image option assembly 분리
3. volume mode decision 분리
4. volume runtime helper 분리
5. container runtime helper 분리
6. image build/push runtime helper 분리

## 성공 조건

- 문서가 GitHub에 올라가 있음
- 앞선 pure/runtime slice들이 같이 올라가 있음
- clean VM runtime test가 다시 통과함

---

# Sprint: Runtime Refactor And Test Split

## Goal

The goal of this sprint is to separate runtime-dependent code from pure logic more clearly inside `podbridge5`,
so failures can be classified faster.

The previous sprint established the following:

- reproducible runtime tests on a fresh VM
- automation that syncs the current local worktree into the remote VM
- automated VM lifecycle and log collection

The next requirement is:

- explicit runtime contracts
- wider pure-Go validation coverage
- clearer isolation of runtime failure causes

## Scope

### 1. Lock down the runtime contract

First, make the `executor.sh`, `healthcheck.sh`, `/app` paths, result log paths, and default command explicit as code constants.
The purpose of this step is to reduce string duplication and create a stable baseline for later refactors.

Targets:

- `container.go`
- `executor.go`
- `buildconfig.go`
- related unit tests

Definition of done:

- `/app/executor.sh`, `/app/healthcheck.sh`, `/app/result.log`, `/app/exit_code.log` are managed as shared constants
- healthcheck command parsing is moved into a dedicated pure function
- that logic is covered by unit tests

Status:

- completed

### 2. Separate image option assembly

Next, extract the values assembled before the buildah runtime call inside `image.go` into pure helper functions.
The purpose of this step is to validate build options, push destinations, and default builder options without requiring the full build runtime.

Targets:

- `image.go`
- new pure helpers
- new unit tests

Definition of done:

- build option assembly is moved into a dedicated pure function
- push destination normalization is moved into a dedicated pure function
- default builder/add-copy options are moved into dedicated pure functions
- that logic is covered by unit tests

Status:

- implemented in this turn

### 3. Clarify the unit vs runtime test boundary

The next step is to make the test layers more explicit.

- unit: pure logic, parsing, option composition, error contracts
- runtime: real podman/buildah/storage/socket validation

In this sprint, the first concrete move is to make the healthcheck/executor contract and image option assembly more unit-friendly,
then extend the same pattern into `volume`.

### 3. Separate volume mode decisions

Next, extract the volume mode decision logic from `volume.go` into a pure function before the runtime calls.
The purpose of this step is to validate how `ModeSkip`, `ModeUpdate`, and `ModeOverwrite` resolve against an existing or missing volume without requiring the full runtime path.

Targets:

- `volume.go`
- new pure helper
- new unit test

Definition of done:

- the volume mode decision is moved into a dedicated pure function
- `WriteFolderToVolume` uses that decision result
- that logic is covered by unit tests

Status:

- implemented in this turn

### 4. Clarify the unit vs runtime test boundary

The next step is to make the test layers more explicit.

- unit: pure logic, parsing, option composition, error contracts
- runtime: real podman/buildah/storage/socket validation

In this sprint, the first concrete move is to make the healthcheck/executor contract, image option assembly, and volume mode decisions more unit-friendly,
then continue pushing the runtime-only layer split further.

### 5. Separate container runtime helpers

Next, move the image/container runtime calls mixed directly into `container.go` behind a dedicated helper layer.
The purpose of this step is to let `CreateContainer`, `StartContainer`, and `InspectContainer` focus on flow and contracts,
while the real Podman binding calls and inspect-state mapping sit behind a thinner boundary.

Targets:

- `container.go`
- new runtime helper
- new unit test

Definition of done:

- container exists / image ensure / create / start / inspect are called through runtime helpers
- `CreateContainer`, `StartContainer`, `InspectContainer`, and `handleExistingContainer` use those runtime helpers
- inspect state -> `ContainerStatus` mapping is moved into a dedicated pure function
- the helper and status-mapping logic are covered by unit tests

Status:

- implemented in this turn

### 6. Separate image build/push runtime helpers

Next, move the core build/push path in `image.go` behind a dedicated runtime helper layer.
The purpose of this step is to let `BuildDockerfileContent`, `PushImage`, and `BuildAndPushDockerfileContent` focus on flow and input validation,
while the real buildah/imagebuildah calls sit behind a thinner boundary.

Targets:

- `image.go`
- new runtime helper
- new unit test

Definition of done:

- build Dockerfile / push image calls are performed through runtime helpers
- temp Dockerfile writing and build/push orchestration are moved into helper functions
- `BuildDockerfileContent`, `PushImage`, and `BuildAndPushDockerfileContent` use the runtime helpers
- the helper orchestration logic is covered by unit tests

Status:

- implemented in this turn

### 5. Prepare the following slice

After this sprint, the next immediate follow-up work should be:

- pushing the runtime-only layer split further in `image.go` and `volume.go`
- specifying dry-run and timeout behavior more concretely

## Work started in this turn

The implementation started from this document up to this turn covered these six slices:

1. lock down the runtime contract
2. separate image option assembly
3. separate volume mode decisions
4. separate volume runtime helpers
5. separate container runtime helpers
6. separate image build/push runtime helpers

## Success criteria

- the document is published to GitHub
- the current pure/runtime slices are published with it
- the clean-VM runtime test passes again
