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

### 4. 다음 후속 작업 준비

이 스프린트가 끝나면 후속으로 바로 이어질 작업은 다음입니다.

- `volume.go`의 mode/overwrite/update 판단 로직 분리
- `image.go`와 `volume.go`의 runtime-only 레이어 분리 심화
- dry-run / timeout 기능 구체화

## 이번 턴에서 바로 시작한 작업

이 문서를 기준으로 이번 턴에서 바로 진행한 구현은 아래 두 가지입니다.

1. runtime contract 고정
2. image option assembly 분리

## 성공 조건

- 문서가 GitHub에 올라가 있음
- 첫 두 개의 pure-logic slice가 같이 올라가 있음
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

### 4. Prepare the following slice

After this sprint, the next immediate follow-up work should be:

- separating mode and overwrite/update decision logic in `volume.go`
- pushing the runtime-only layer split further in `image.go` and `volume.go`
- specifying dry-run and timeout behavior more concretely

## Work started in this turn

The implementation started from this document in this turn covered these two slices:

1. lock down the runtime contract
2. separate image option assembly

## Success criteria

- the document is published to GitHub
- the first two pure-logic slices are published with it
- the clean-VM runtime test passes again
