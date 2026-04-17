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

상태:

- 완료

### 2. Image option assembly 분리

상태:

- 완료

### 3. Volume mode decision 분리

상태:

- 완료

### 4. Volume runtime helper 분리

상태:

- 완료

### 5. Container runtime helper 분리

상태:

- 완료

### 6. Image build/push runtime helper 분리

상태:

- 완료

### 7. Image builder runtime helper 분리

상태:

- 완료

### 8. Image export runtime helper 분리

상태:

- 완료

### 9. BuildConfig 조립 helper 분리

상태:

- 완료

### 10. Verification split

상태:

- 완료

### 11. Runtime preflight diagnostics 강화

상태:

- 완료

### 12. Runtime init policy hardening 시작

이번 단계에서는 runtime 초기화 정책을 코드와 문서에 명시적으로 고정합니다.
핵심은 `CONTAINER_HOST` 우선순위, `Init()` / `Shutdown()` 재시도 가능 계약,
그리고 분류 가능한 runtime init 에러를 도입하는 것입니다.

대상:

- `connect_linux_5.go`
- `rootless.go`
- 신규 runtime init helper / unit test
- runtime 정책 문서

완료 기준:

- Linux connection URI 우선순위가 코드와 문서에서 일치함
- `runtime-host-check`와 실제 Go runtime 해석 규칙이 일치함
- `ErrRuntimeConnectionUnavailable`, `ErrRuntimeStoreUnavailable`, `ErrRuntimeNotInitialized`가 도입됨
- `Init()` 실패 후 환경 수정 뒤 재시도가 가능함
- 관련 helper가 unit test로 검증됨

상태:

- 이번 턴에서 구현 진행

## 후속 작업

다음 단계에서는 아래 항목을 이어서 진행합니다.

- dry-run / timeout 정책 고정
- stable API와 internal helper 경계 문서화
- VM 경로 복구 후 runtime init 정책의 clean VM 재검증
