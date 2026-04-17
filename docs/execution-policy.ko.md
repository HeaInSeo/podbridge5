# Dry-Run 과 Timeout 정책

이 문서는 `podbridge5`의 현재 dry-run 과 timeout 계약을 설명합니다.
핵심은 "무엇이 이미 정책으로 고정되었고, 무엇은 아직 범위 밖인가"를 명확히 하는 것입니다.

## Dry-Run 정책

현재 `podbridge5`에서 문서화된 dry-run 정책은 `buildah.AddAndCopyOptions` 경로에 한정합니다.

- 기본값: `DefaultAddAndCopyDryRun = false`
- 의미: 기본 동작은 builder state를 실제로 변경하는 실행 경로
- override: 필요하면 `WithDryRun(true)` 또는 내부 helper의 explicit dry-run 경로 사용

이 기본값을 false로 고정한 이유는, 현재 공개 함수들의 주된 목적이 실제 이미지 조립과 스크립트 주입이기 때문입니다.
즉, dry-run은 opt-in 정책이고 기본 동작은 mutation입니다.

현재 범위에서 dry-run이 의미하는 것은 다음뿐입니다.

- add/copy 계열 builder 조작의 no-op 의도 전달

현재 범위 밖:

- image build 전체 dry-run
- push dry-run
- export dry-run
- container lifecycle dry-run
- volume write dry-run

즉, 지금의 dry-run은 라이브러리 전체 실행 정책이 아니라 add/copy 옵션 표면의 제한된 계약입니다.

## Timeout 정책

현재 `podbridge5`에서 공통 정책으로 고정한 timeout은 healthcheck 계약입니다.

기본값:

- `DefaultHealthcheckInterval = 30s`
- `DefaultHealthcheckRetries = 3`
- `DefaultHealthcheckTimeout = 5s`
- `DefaultHealthcheckStartPeriod = 0s`
- 최소 timeout: `MinHealthcheckTimeout = 1s`

의도는 단순합니다.

- healthcheck timeout은 너무 짧으면 의미가 없으므로 1초 미만을 허용하지 않음
- interval은 `disable` 또는 `0`으로 비활성화 가능
- start period는 음수를 허용하지 않음

`ParseHealthcheckConfig(...)`는 이 공통 helper를 사용해 같은 규칙으로 파싱합니다.

## 현재 범위 밖

이 문서가 다루지 않는 timeout은 다음입니다.

- build timeout
- push timeout
- export timeout
- container start/stop timeout의 공통 정책
- context cancellation 정책의 라이브러리 전역 표준화

즉, 현재 timeout 정책은 healthcheck 계약까지만 고정된 상태입니다.
나머지는 후속 Sprint 2 작업에서 public API 경계와 함께 정리해야 합니다.
