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

상태:

- 완료

### 13. Dry-run / timeout policy hardening 시작

이번 단계에서는 현재 라이브러리가 실제로 지원하는 dry-run 과 timeout 범위를 문서와 코드에서 고정합니다.
핵심은 add/copy dry-run 기본값, healthcheck timeout 기본값과 최소값,
그리고 "아직 library-wide policy가 아닌 영역"을 분명히 적는 것입니다.

대상:

- 신규 execution policy helper / unit test
- `runtime_contract.go`
- `image_options.go`
- execution policy 문서

완료 기준:

- `DefaultAddAndCopyDryRun`가 코드 상수로 고정됨
- healthcheck interval/retries/timeout/start period 기본값이 코드 상수로 고정됨
- healthcheck timeout 최소값이 공통 helper로 검증됨
- `ParseHealthcheckConfig(...)`가 공통 helper를 사용함
- dry-run / timeout 범위와 비범위가 문서에 명시됨

상태:

- 이번 턴에서 구현 진행

## 후속 작업

다음 단계에서는 아래 항목을 이어서 진행합니다.

- stable API와 internal helper 경계 문서화
- build/push/export option 정책 문서화
- VM 경로 복구 후 runtime policy의 clean VM 재검증
