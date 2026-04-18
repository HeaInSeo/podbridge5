# Implementation Priorities

이 문서는 현재 `podbridge5`에서 구현적으로 더 보강해야 할 항목을 우선순위 기준으로 정리합니다.
핵심은 "지금 무엇을 먼저 고쳐야 전체 완성도가 가장 크게 올라가는가"입니다.

## 현재 판단

`podbridge5`는 이미 내부 runtime utility 라이브러리로는 충분히 의미 있는 수준입니다.
runtime helper 분리, 테스트 계층 분리, runtime init policy, dry-run / timeout policy의 기초 정리는 진행됐습니다.

다만 아직 완성형 라이브러리라고 보기는 어렵습니다.
현재는 새로운 기능을 크게 늘리는 것보다,
기존 기능의 실행 계약과 외부 surface를 더 명확히 고정하는 편이 우선입니다.

## 우선순위

### 1. Stable API 와 Internal Helper 경계 정리

가장 먼저 보강해야 할 항목입니다.

현재는 runtime helper가 많이 분리됐지만,
외부 사용자가 안정적으로 의존해도 되는 API와 내부 리팩터용 helper가 아직 완전히 분리된 상태는 아닙니다.
이 경계가 흐리면 이후 기능 추가 때 다시 debt가 쌓입니다.

보강 방향:

- public API 후보 목록 정리
- internal helper 성격 함수 목록 정리
- README / docs에서 외부 surface 명시
- 필요하면 파일명, 함수명, package-level comment 정리

완료 기준:

- 어떤 함수가 stable contract인지 설명 가능함
- internal helper를 외부 사용 예제로 노출하지 않음
- 다음 기능 추가가 helper debt를 다시 키우지 않음

### 2. Timeout 정책 확대

현재 공통 정책으로 고정된 timeout은 healthcheck 계약이 사실상 전부입니다.
실제 운영에서는 아래 경로들도 timeout 정책이 필요합니다.

- image build
- image push
- image export
- container start / stop
- volume write / read

지금은 이 부분을 호출자가 `context`로 알아서 제어해야 하는 경우가 많아서,
라이브러리 차원의 실행 계약이 아직 약합니다.

보강 방향:

- timeout이 필요한 public operation 목록 정리
- context-only 정책으로 둘지, explicit option으로 둘지 결정
- 기본값을 둘지, caller required로 둘지 결정
- 문서와 테스트에 같은 계약 반영

완료 기준:

- timeout 지원 범위가 문서에 명확함
- build/push/container/volume 경로의 timeout 해석이 일관됨
- timeout 실패가 호출부에서 예측 가능함

### 3. Dry-Run 범위 결정

현재 dry-run 정책은 add/copy 옵션 표면까지만 고정되어 있습니다.
이 상태는 나쁘지 않지만, 이름만 보면 라이브러리 전체 실행 정책처럼 보일 수 있습니다.

즉, 다음 둘 중 하나를 선택해야 합니다.

- dry-run을 실제로 더 넓혀서 build/push/container/volume에도 적용
- 아니면 현재 범위를 더 명확히 제한하고 public surface에서도 좁게 유지

보강 방향:

- dry-run이 필요한 operation 목록 정리
- 각 operation에서 dry-run이 의미하는 바를 정의
- no-op 검증 결과를 어떻게 반환할지 결정
- 문서와 테스트를 operation별로 맞춤

완료 기준:

- dry-run이 어디까지 지원되는지 혼동이 없음
- 현재 범위를 넘는 기대를 문서가 만들지 않음
- 필요 시 후속 확장이 가능한 형태로 계약이 정리됨

### 4. 단계별 에러 분류 강화

runtime init 쪽은 `ErrRuntimeConnectionUnavailable`, `ErrRuntimeStoreUnavailable`, `ErrRuntimeNotInitialized`가 들어가면서 좋아졌습니다.
하지만 build/push/container/volume 단계는 여전히 "어디서 실패했는지"를 기계적으로 분류하기 어려운 경우가 많습니다.

상위 프로젝트에서 쓰려면 이 부분이 중요합니다.
특히 orchestration 레이어는 실패 위치를 알아야 재시도와 사용자 메시지를 분기할 수 있습니다.

보강 방향:

- build / push / export / container / volume 단계별 sentinel error 후보 정리
- `errors.Is` / `errors.As` 기준을 통일
- 로그 메시지와 에러 wrapping 규칙 정리

완료 기준:

- 실패 지점을 상위 프로젝트가 분기 가능함
- retryable / non-retryable 판단에 도움되는 에러 모델이 있음
- unit test에서 wrapping 규칙을 검증 가능함

### 5. Clean VM Runtime 재검증

이건 새 구현이라기보다, 지금까지 정리한 정책을 runtime 환경에서 다시 확인하는 단계입니다.
VM이 복구되면 가장 먼저 해야 합니다.

중요한 이유는 단순합니다.
unit 경로는 많이 강해졌지만,
실제 Podman socket / store / buildah 환경에서는 clean VM 검증이 최종 기준이기 때문입니다.

재검증 대상:

- runtime init policy
- execution policy
- test-runtime / vm-test-runtime 경로 일관성
- host preflight와 실제 runtime 경로 일치 여부

완료 기준:

- fresh VM 기준에서 정책 문서와 실제 동작이 일치함
- 환경 찌꺼기 없이 재현 가능한 결과가 나옴
- VM 경로 보류 항목을 다시 닫을 수 있음

## 추가 구현 사항

우선순위 외에 뒤이어 볼 수 있는 보강 항목도 있습니다.

### build/push/export option 문서화

현재는 기본 정책이 일부만 정리돼 있습니다.
실제 사용자가 각 경로에서 무엇을 조절할 수 있는지 문서가 더 필요합니다.

### examples / quickstart 추가

현재 README와 runtime docs는 좋아졌지만,
새 사용자가 바로 따라 할 수 있는 예제는 아직 부족합니다.

### support matrix / prerequisites 문서화

어떤 환경에서 안정적으로 동작하는지,
어떤 패키지가 필요한지,
어떤 환경은 비권장인지가 더 명확해질 필요가 있습니다.

### release checklist

외부 재사용 라이브러리 수준으로 가려면,
릴리스 전에 무엇을 확인해야 하는지 체크리스트가 있어야 합니다.

## 추천 진행 순서

추천 순서는 아래와 같습니다.

1. stable API / internal helper 경계 문서화
2. timeout 정책 확대
3. dry-run 범위 결정
4. 단계별 에러 분류 강화
5. VM 복구 후 clean VM 재검증
6. examples / support matrix / release checklist 정리

## 결론

현재 `podbridge5`에서 가장 중요한 것은 기능을 더 많이 넣는 것이 아닙니다.
이미 있는 기능을 "설명 가능하고, 예측 가능하고, 검증 가능한 상태"로 만드는 것입니다.

즉, 다음 보강의 중심은 아래 세 가지입니다.

- 외부 contract 명확화
- 실행 정책 일관성 강화
- runtime 검증 신뢰도 회복

이 세 가지가 정리되면, 그 다음 기능 확장은 지금보다 훨씬 안전하게 갈 수 있습니다.
