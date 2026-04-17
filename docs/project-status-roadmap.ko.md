# Project Status And Sprint Roadmap

이 문서는 현재 `podbridge5`의 수준을 세 가지 기준으로 정리합니다.

- 완료된 것
- 미완인 것
- 위험한 것

그리고 이 세 기준을 바탕으로 다음 스프린트 일정을 제안합니다.

## 현재 수준 요약

현재 `podbridge5`는 내부 runtime utility 라이브러리로는 이미 충분히 의미 있는 수준입니다.
`container`, `volume`, `image`, `buildconfig`에 걸쳐 runtime 호출과 순수 로직을 상당 부분 분리했고,
clean VM 기반 runtime 검증 경로도 방향이 잡혀 있습니다.

다만 아직 공개 라이브러리 수준의 완성형이라고 보기는 어렵습니다.
가장 큰 이유는 기능 부족보다도 다음 세 가지입니다.

- full runtime 검증의 안정성 부족
- unit / runtime / integration 경계의 운영 수준 분리 미완료
- 안정 API와 내부 helper의 계약 문서화 부족

현재 판단은 아래와 같습니다.

- 내부 사용 기준: 가능
- 상위 프로젝트 의존 라이브러리 기준: 가능
- 외부 재사용 라이브러리 기준: 아직 보완 필요

## 1. 완료된 것

### 구조

- runtime contract 상수 정리
- healthcheck/parser 분리
- image option assembly 분리
- volume mode decision 분리
- volume/container/image/buildconfig 전반의 runtime helper 분리
- build/push/export/builder 조작 경로의 helper 분리

### 테스트 방향

- 순수 helper를 unit test로 검증하는 구조 확보
- remote clean VM 기반 runtime test 자동화 확보
- 로그 수집 경로 확보
- worktree sync 기반 fresh VM 재검증 경로 확보

### 문서 방향

- README를 프로젝트 소개 중심으로 재구성
- runtime 검증 문서를 별도 분리
- 한국어/영문 문서 분리
- 리팩터 스프린트 진행 문서 유지

### 평가

이 영역은 현재 **기반 공사 완료**로 봐도 됩니다.
즉, 무질서하게 직접 runtime 호출이 섞여 있던 상태는 많이 벗어났고,
앞으로는 구조 위에서 안정성, 정책, 계약을 다듬는 단계로 넘어갈 수 있습니다.

## 2. 미완인 것

### 검증 체계

- unit / runtime / integration 타깃을 Makefile 수준에서 더 명확히 나눌 필요
- clean VM full 검증이 항상 안정적으로 끝나는 상태는 아님
- runtime 실패를 더 빠르게 분류할 수 있는 진단 출력이 부족함

### API와 계약

- 어떤 API를 stable surface로 볼지 아직 명확하지 않음
- legacy 성격 함수와 내부 helper의 구분이 더 필요함
- 일부 함수는 여전히 역할이 넓고, naming도 정리 여지가 있음

### 기능 정책

- dry-run / timeout 정책은 아직 문서와 코드가 충분히 정리되지 않음
- buildah/podman/store 초기화 정책을 더 명시할 필요가 있음
- export/save/build path의 옵션 정책을 더 문서화할 필요가 있음

### 운영 문서

- 외부 사용자 관점의 사용 예제가 부족함
- 지원 환경, 비지원 환경, 기대 실행 모델에 대한 계약 문서가 더 필요함
- release checklist 문서가 없음

### 평가

이 영역은 현재 **완성도 보강 단계**입니다.
지금 멈추면 내부 사용은 가능하지만, 장기 유지보수 비용이 높아질 수 있습니다.

## 3. 위험한 것

### 검증 환경 리스크

- clean VM 검증이 apt mirror/network 상태에 영향을 받음
- local full test는 podman socket 준비 여부에 크게 영향받음
- 즉, 코드와 무관한 환경 이슈가 검증 신뢰도를 흔들 수 있음

### API 확장 리스크

- 지금 상태에서 기능을 더 빠르게 추가하면 구조 정리 전 debt가 다시 쌓일 수 있음
- runtime helper 분리를 끝내기 전에 public API를 넓히면 나중에 계약 변경 비용이 커질 수 있음

### 유지보수 리스크

- helper는 늘었지만 “무엇이 외부 계약인지”가 아직 완전히 고정되진 않음
- 테스트는 늘었지만, 전체 경로가 한 번에 항상 검증되는 상태는 아직 아님

### 평가

이 영역은 현재 **출시 전 리스크 관리 단계**입니다.
지금 가장 위험한 것은 기능이 없는 것이 아니라,
검증 불안정성과 계약 미고정 상태에서 사용 범위를 넓히는 것입니다.

## 스프린트 일정 제안

기준은 단순합니다.

- Sprint 1: 미완인 것 중 검증 체계 마무리
- Sprint 2: 위험한 것 중 운영/계약 리스크 축소
- Sprint 3: 공개 가능 수준 문서와 릴리스 기준 정리

### Sprint 1. Verification Split

기간:

- 1주

목표:

- unit / runtime / integration 타깃을 명확히 분리
- Makefile과 문서에서 검증 경로를 고정
- clean VM 검증 실패 시 원인 분류 로그를 강화

작업:

- `make test-unit`, `make test-runtime`, `make test-runtime-integration` 정리
  - 현재 턴에서 착수
- runtime-only 테스트 파일/패턴을 더 명확히 구분
- VM 검증 시 단계별 로그와 실패 지점 표기 강화
- 로컬 `test-runtime` preflight 메시지 강화
- local fast path와 clean VM path의 계약 문서 정리

완료 기준:

- 개발자가 어떤 테스트를 언제 돌려야 하는지 명확함
- unit과 runtime 실패가 로그 수준에서 구분됨
- clean VM 테스트 실패 시 환경/코드 원인 분류가 더 쉬워짐

보류 항목:

- VM 검증 단계별 로그 포맷 정리
- VM 경로의 최종 재현성 확인

보류 조건:

- `100.123.80.48` 측 VM 작업이 끝난 뒤 재개

### Sprint 2. Runtime Policy Hardening

기간:

- 1주

목표:

- dry-run / timeout / runtime init 정책 고정
- build/push/export 경로의 운영 계약 명시
- helper와 public API 경계 더 정리

작업:

- dry-run / timeout 정책 설계 및 적용
- runtime init 및 prerequisite 계약 문서화
- stable API 후보와 internal helper 구분
- legacy/중복 경로 정리

완료 기준:

- 실행 정책이 문서와 코드에서 일치함
- 주요 public API의 계약이 설명 가능함
- 내부 helper와 외부 surface가 더 명확해짐

### Sprint 3. Release Readiness

기간:

- 1주

목표:

- 외부 공유 가능한 수준의 문서와 release 기준 확보
- 지원 환경, 제한 사항, 예제, 체크리스트 정리

작업:

- quickstart / examples 문서 추가
- support matrix 또는 prerequisites 문서 추가
- release checklist 작성
- known limitations 문서 작성

완료 기준:

- 새 사용자가 README와 docs만 보고 진입 가능함
- 유지보수자가 release 전에 무엇을 확인해야 하는지 명확함
- “내부 전용 도구”에서 “재사용 가능한 라이브러리”로 넘어갈 최소 문서가 갖춰짐

## 결론

현재 `podbridge5`는 멈춰도 되는 상태는 아닙니다.
하지만 방향이 틀린 상태도 아닙니다.

정리하면:

- 기반 구조: 충분히 좋아짐
- 실사용 가능성: 이미 있음
- 완성도: 아직 보강 필요
- 다음 우선순위: 기능 추가보다 검증/계약/운영 안정화

즉, 다음 세 스프린트는 새로운 기능을 크게 늘리는 것보다,
지금 만든 구조를 **검증 가능하고 설명 가능한 상태로 고정하는 과정**으로 잡는 것이 맞습니다.
