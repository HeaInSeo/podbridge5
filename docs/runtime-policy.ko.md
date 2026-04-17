# Runtime 초기화 정책

이 문서는 `podbridge5`가 Podman runtime 연결과 store 초기화를 어떻게 결정하는지 설명합니다.
`Sprint 2`의 목적은 이 정책을 코드와 문서에서 같은 방식으로 고정하는 것입니다.

## 연결 우선순위

`podbridge5`는 Linux에서 Podman 연결 URI를 다음 순서로 결정합니다.

1. `CONTAINER_HOST`
2. `XDG_RUNTIME_DIR`
3. rootless 기본값 `/run/user/<uid>/podman/podman.sock`
4. root 기본값 `/run/podman/podman.sock`

즉, `CONTAINER_HOST`가 설정되어 있으면 자동 socket 추정보다 항상 우선합니다.
이는 host-side test와 remote clean-VM test가 같은 정책을 따르도록 하기 위한 것입니다.

## Init / Shutdown 계약

초기화 관련 public entrypoint는 다음입니다.

- `Init()`
- `InitWithContext(ctx)`
- `Shutdown()`
- `ReexecIfNeeded()`

현재 계약은 아래와 같습니다.

- `Init()` / `InitWithContext(ctx)`는 Podman 연결과 store 준비를 함께 수행합니다.
- 초기화가 실패하면 에러를 반환하고, 성공 상태로 고정하지 않습니다.
- 즉, 환경을 고친 뒤 다시 `Init()`을 호출하면 재시도할 수 있습니다.
- `Shutdown()`은 초기화가 끝난 경우에만 유효합니다.
- `Shutdown()` 뒤에는 다시 `Init()`이 가능합니다.

## 분류 가능한 에러

런타임 초기화 오류는 `errors.Is`로 분류할 수 있게 유지합니다.

- `ErrRuntimeConnectionUnavailable`
- `ErrRuntimeStoreUnavailable`
- `ErrRuntimeNotInitialized`

의도는 단순합니다.

- Podman socket / connection 문제인지
- store 초기화 문제인지
- 아직 초기화하지 않은 상태인지

를 호출부에서 빠르게 구분할 수 있어야 합니다.

## 테스트와의 관계

로컬 preflight와 코드 정책은 같은 기준을 따라야 합니다.

- `make runtime-host-check`
- `make test-runtime`
- `make test-runtime-integration`

`runtime-host-check`도 같은 순서로 해석합니다.

1. `CONTAINER_HOST`
2. `XDG_RUNTIME_DIR`
3. 기본 Podman socket

즉, preflight가 통과한 호스트에서 코드가 다른 socket을 보지 않도록 맞추는 것이 목적입니다.

## 범위 밖

이 문서가 다루지 않는 것은 다음입니다.

- build/push/export별 세부 option 정책
- healthcheck timeout parsing 정책
- clean VM lifecycle 자체

이 항목들은 다른 문서에서 따로 다룹니다.
