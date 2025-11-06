# Daylog 기술 아키텍처 v2.1

## 1. 제품 범위 요약
- 비전: 사용자의 하루를 자동으로 기록하고 익명화된 인사이트를 제공하며, 유사한 패턴을 가진 사람들과 연결해 주는 라이프스타일 SNS.
- 핵심 축: 자동 라이프 로깅, 개인정보를 보호하는 라벨링, 소셜 및 커뮤니티 상호작용, AI 분석과 추천 기능, 프리미엄 접근 제어.
- 타깃 플랫폼: 모바일 퍼스트(React Native 기반 iOS/Android)와 인사이트·커뮤니티 기능을 위한 반응형 웹 대시보드(Next.js).

## 2. 상위 수준 아키텍처
```
+--------------------------------------------------------------+
|                      클라이언트 애플리케이션                 |
|  - React Native 모바일 앱      - Next.js 웹 대시보드         |
|  - 오프라인 캐시(MMKV/IndexDB) - 푸시 알림                   |
+-----------------------------+--------------------------------+
                              | GraphQL(페더레이션) / WebSocket
+-----------------------------+--------------------------------+
|            API Gateway & BFF (Apollo Gateway)                |
|   인증, 레이트 리밋, 클라이언트 ACL 적용                    |
+-------------+---------------------------+--------------------+
              | gRPC / REST / 이벤트 스트림
+-------------+---------------------------+--------------------+
|                   도메인 마이크로서비스                     |
|  수집            타임라인           소셜 피드               |
|  분류            피드백 학습        커뮤니티                |
|  라벨 서비스     분석               추천                     |
|  결제 & 권한 관리                                          |
+------+-------+-------+-------+--------------+----------------+
       |       |       |       |              |
+------+-------+-------+-------+--------------+----------------+
|                        데이터 레이어                         |
|  Aurora PostgreSQL, DynamoDB, Redis, OpenSearch, Neo4j       |
|  S3 데이터 레이크(원시 이벤트, 피처 저장)                    |
+------+--------------------------------------------------------+
       |
+------+--------------------------------------------------------+
|                 분석 & 머신러닝 플랫폼                       |
|  피처 스토어, SageMaker 파이프라인, Glue/Spark, Flink        |
+---------------------------------------------------------------+
```

### 배포 및 운영
- 인프라: AWS 기반. Terraform과 AWS CDK로 VPC, EKS, Aurora, MSK, OpenSearch 등을 프로비저닝합니다.
- 런타임: 무상태 서비스는 EKS(Kubernetes)에 배포하고, 데이터베이스 등은 매니지드 서비스를 활용합니다.
- CI/CD: GitHub Actions로 빌드·테스트 후 ArgoCD로 개발/스테이징/프로덕션에 블루-그린 방식 배포합니다.
- 관측성: OpenTelemetry(OTLP)를 Prometheus, Tempo, Loki로 수집하고 CloudWatch, Datadog으로 이상을 탐지합니다.
- 시크릿 관리: AWS Secrets Manager를 사용해 키를 자동으로 로테이션합니다.

## 3. 모듈 책임

### 3.1 수집 및 타임라인
- Ingestion 서비스: 기기 웹훅(스크린타임, 위치, 캘린더, 건강 데이터)을 JSON Schema/Protobuf로 검증하고 표준화된 `ActivityEvent`를 Kafka 토픽 `activity.raw`에 발행합니다.
- Timeline Aggregator: 이벤트를 소비하며 지오펜스 메타데이터로 보강하고 겹치는 구간을 병합합니다. 결과 타임라인을 Aurora `timeline_entries` 테이블에 저장하고 최신 1일 분량은 Redis에 캐싱합니다. 분석 완료 시 알림을 트리거합니다.

### 3.2 AI 분류 및 피드백 루프
- Activity Classifier: FastAPI + ONNX Runtime 기반 실시간 추론 서비스로, 피처 스토어의 시간·위치·앱 사용량 정보를 활용해 활동 카테고리를 분류합니다.
- Feedback Trainer: 사용자 수정(`activity_feedback`)을 수집해 매일 밤 SageMaker 파이프라인으로 재학습을 실행합니다. MLflow 모델 레지스트리로 버전을 관리하고 LaunchDarkly 카나리 배포로 점진적으로 롤아웃합니다.

### 3.3 라벨 및 신원 서비스
- `/소속`, `/신분`, `/관심사` 등 구조화된 라벨을 관리합니다.
- 이메일/도메인 인증(SES + Lambda)으로 검증된 라벨을 표시합니다.
- OpenSearch 인덱스(`users_public`, `labels`)를 최신 상태로 유지해 익명성을 확보하면서 탐색 기능을 제공합니다.

### 3.4 소셜 및 커뮤니티
- Social Feed 서비스: 타임라인 공유, 팔로우 라벨, 추천 사용자 콘텐츠를 조합한 개인화 피드를 생성합니다. DynamoDB Streams로 팬아웃하고 Redis 정렬 세트로 순위를 관리합니다.
- Community 서비스: 그룹, 멤버십, 게시글, 챌린지를 관리합니다. Pro 전용 보드는 권한 정보를 기반으로 게이트웨이와 서비스 양단에서 검증합니다. Perspective API 등을 활용한 모더레이션 훅과 챌린지 알림을 지원합니다.

### 3.5 분석 및 추천
- Analytics 서비스: Presto/Trino로 큐레이션된 데이터셋을 조회해 Free 이용자에게 라벨 기반 통계를 제공합니다. Pro 보고서를 비동기로 생성해 S3에 저장하고 서명 URL을 배포합니다.
- Recommendation 엔진: GraphSAGE 기반 임베딩을 Neo4j에 적재해 일과 리듬, 위치, 관심사를 고려한 유사 사용자 추천을 수행합니다. GraphQL 구독과 푸시 알림으로 결과를 전달합니다.

### 3.6 빌링 및 접근 제어
- Stripe Billing으로 구독, 결제 의도, 인보이스를 처리합니다.
- 권한 정보는 Aurora `user_entitlements`에 저장하고 Redis에 캐싱하며, JWT 스코프에 `tier:free|pro`를 포함합니다.
- 유예 기간, 다운그레이드, Stripe 웹훅 기반 상태 동기화를 지원합니다.

## 4. 데이터 모델 (Aurora PostgreSQL)

| 테이블 | 목적 | 주요 컬럼 |
|-------|------|-----------|
| users | 기본 사용자 프로필 | id(UUID PK), email, status, tier, created_at |
| user_settings | 개인정보 공개/알림 설정 | user_id, data_visibility_level, timezone |
| user_labels | 구조화된 라벨 | user_id, label_key, label_value, is_verified, verified_at |
| devices | 연동된 디바이스 | user_id, device_id, platform, last_seen |
| activity_events | 분류 전 원시 이벤트 | event_id, user_id, source, timestamp_start, timestamp_end, metadata JSONB |
| timeline_entries | 통합 타임라인 블록 | timeline_id, user_id, category, confidence, geo_context, source_event_ids array |
| activity_feedback | 사용자 수정 기록 | feedback_id, timeline_id, user_id, old_category, new_category, note |
| social_posts | 공유된 타임라인 스냅샷 | post_id, user_id, timeline_id, visibility, like_count |
| social_reactions | 반응 데이터 | reaction_id, post_id, user_id, type, created_at |
| comments | 댓글 | comment_id, post_id, user_id, body, moderation_state |
| communities | 커뮤니티/게시판 | community_id, access_level, title, is_pro_only |
| community_memberships | 멤버십 관계 | community_id, user_id, role, joined_at |
| challenges | 게이미피케이션 챌린지 | challenge_id, community_id, target_metric, start_at, end_at |
| analytics_reports | Pro 리포트 메타데이터 | report_id, user_id, report_type, status, s3_uri |
| user_entitlements | 구독 상태 | user_id, tier, renewal_date, status, stripe_subscription_id |

추가 저장소:
- DynamoDB `user_activity_cache`: 최근 7일 이벤트를 보관해 저지연 조회를 지원합니다.
- OpenSearch 인덱스 `users_public`, `timelines_public`: 익명화된 검색 및 필터링에 사용합니다.
- Neo4j 그래프: 사용자·라벨 노드와 유사도, 상호작용 엣지를 관리합니다.
- S3 데이터 레이크: `activity/raw/year=...` 형태로 원시 이벤트를 파티셔닝하고, 큐레이션 데이터셋과 모델 아티팩트를 보관합니다.

## 5. API 설계
- 게이트웨이는 GraphQL(페더레이션 스키마)을 클라이언트에 제공하고, Apple Health·Google Fit 등 외부 연동에는 REST 웹훅을 제공합니다.
- GraphQL 예시:
  - `Query.timeline(day: Date!): [TimelineEntry!]`
  - `Query.labelInsights(label: LabelInput!): LabelAggregate!`
  - `Mutation.submitFeedback(timelineId: ID!, correction: FeedbackInput!)`
  - `Mutation.joinCommunity(communityId: ID!)`
  - `Mutation.generateProReport(type: ReportType!)`
  - `Subscription.matchRecommendations: [UserMatch!]`
- 인증: Amazon Cognito 또는 Auth0로 OAuth, Apple, Google 로그인을 지원합니다. 액세스 토큰에는 이용 등급, 스코프, 검증된 라벨 정보가 포함되며, 서비스 간 통신은 IAM 역할과 mTLS를 사용합니다.

## 6. AI 및 ML 전략
- 모델 구성:
  - 시간 기반 Transformer 또는 Bi-LSTM을 활용한 활동 분류 모델.
  - 타임라인 시퀀스에 대한 대조 학습 기반 유사도 임베딩 모델.
  - Pro 리포트를 위한 목표 달성도 예측 모델(Prophet/XGBoost).
- 피처 엔지니어링:
  - AWS Glue/Spark로 배치 피처를 계산해 Feast에 등록합니다.
  - Flink로 스트리밍 피처(앱 사용량 롤링 통계, 위치 히트맵)를 계산해 Redis에 적재합니다.
- 지속 학습:
  - 피드백 이벤트를 큐에 적재해 검토합니다.
  - 야간 재학습 시 홀드아웃 데이터로 자동 평가합니다.
  - 전체 배포 전 사용자 5% 대상 카나리 테스트를 적용하고 MLflow 대시보드로 드리프트를 모니터링합니다.

## 7. 프라이버시 및 컴플라이언스
- 데이터 소스별로 명시적 동의를 수집하며, 공개 옵션은 전체 비공개, 통계만 공개, 익명화된 타임라인 공유로 구분합니다.
- 라벨 기반 공개 통계에는 라플라스 노이즈를 적용해 차등 프라이버시를 구현합니다.
- GDPR/CCPA 대응: 이용자 데이터 열람·삭제 요청을 자동화하고 Outbox 패턴으로 캐시와 검색 인덱스까지 연쇄 삭제합니다.
- 감사를 위해 append-only S3 버킷과 Glacier Vault Lock을 사용합니다.

## 8. 접근 제어: Free vs Pro
- API Gateway에서 JWT 스코프를 검증하고, 서비스 내부에서는 미들웨어로 권한 정보를 재확인합니다.
- Free 등급은 최근 30일 데이터와 비교 횟수를 제한하며, Pro는 전체 히스토리, 무제한 비교, Pro 커뮤니티, 고급 분석·추천을 제공합니다.
- 한계치에 근접하면 업그레이드 유도 메시지를 노출하고, 결제 완료 후 서버에서 권한을 확정합니다.
- LaunchDarkly/Unleash로 기능 플래그를 관리해 점진적으로 롤아웃합니다.

## 9. 운영 상 고려 사항
- 보안: AWS WAF, 레이트 리밋, 민감 위치 정보 필드 암호화, 서비스 간 mTLS, 이상 탐지 알림을 적용합니다.
- 확장성: 이벤트 기반 수집 파이프라인은 인터랙티브 API와 독립적으로 확장됩니다. Kafka 파티션을 `user_id` 기준으로 분할하고 Redis 클러스터링으로 읽기 부하를 분산합니다.
- 복원력: 수집 API는 멱등성을 유지하고, 잘못된 이벤트는 지수 백오프로 재시도하며 DLQ로 분류합니다. Envoy 기반 서킷 브레이커를 적용합니다.
- 테스트 전략: GraphQL 계약 테스트, 분류 정확도 검증용 합성 데이터 파이프라인, 타임라인·분석 API를 대상으로 한 k6 부하 테스트를 수행합니다.

## 10. 로드맵 단계
1. MVP(Free 핵심): 수집, 타임라인, 규칙 보조 분류기, 라벨 서비스, 소셜 피드, 커뮤니티 기본 기능, Free 분석.
2. Pro 티어 출시: Stripe 연동, 권한 관리, Pro 분석 리포트, Pro 커뮤니티, 강화된 프라이버시 제어.
3. 고급 AI: 추천 엔진, 유사도 그래프, 목표 예측, 자동화된 지속 학습 파이프라인.
4. 생태계 확장: 서드파티 개발자 API, 파트너용 코호트 인사이트 대시보드.

## 11. 즉시 수행해야 할 다음 단계
- 정의한 스키마에 맞춰 ERD와 마이그레이션(Prisma 또는 Flyway)을 확정합니다.
- Terraform 모듈로 기반 인프라를 구성하고 EKS 클러스터를 부트스트랩합니다.
- React Native용 수집 SDK를 제작해 백그라운드 데이터 수집과 동의 플로우를 구현합니다.
- 라벨 데이터를 축적하기 전까지 결정론적 규칙을 포함한 베이스라인 분류기를 배포합니다.
- 프라이버시 제약과 시각화를 검증하기 위한 합성 데이터 기반 분석 파이프라인을 초기화합니다.

## 12. 현재 구현 스냅샷 (2024-Q4)
- 공통 Go 패키지(`services/common`)로 설정 로더, 구조화 로깅, Postgres 커넥션 풀, Kafka 퍼블리셔/컨슈머 초안을 제공.
- Ingestion 서비스는 활동 이벤트 수신 시 Postgres 저장 및 Kafka 퍼블리시, 헬스/레디 체크 엔드포인트를 지원.
- Timeline 서비스는 Kafka에서 활동 이벤트를 소비해 `timeline_entries`에 업서트하고, 사용자의 타임라인 조회 API를 Postgres 기반으로 응답.
- Label 서비스는 사용자 라벨의 CRUD 스켈레톤을 갖추고 `/readyz` 헬스 체크와 GraphQL 게이트웨이를 통한 라벨 업서트를 지원.
- Social Feed 서비스는 Postgres 기반 피드 조회/작성, Kafka 이벤트 발행, `/readyz` 헬스 체크를 제공.
- Community 서비스는 커뮤니티 생성·목록·가입 API를 Postgres에 연결한 상태로 노출.
- Billing 서비스는 Stripe 웹훅 시그니처 검증, `user_entitlements` 갱신, REST 조회 API와 `/readyz` 헬스 체크를 구현.
- GraphQL 게이트웨이는 서비스 엔드포인트를 환경변수로 주입받고, 타임라인·라벨·피드·커뮤니티·결제 조회와 라벨/피드/커뮤니티 Mutation을 노출하며, 간단한 헤더 기반 인증/티어 검증 로직과 upstream 오류 리포팅을 포함.
- Expo 기반 모바일 앱(`app/`)이 게이트웨이를 호출해 타임라인/피드/커뮤니티/결제 상태를 확인하고, 사용자 ID·티어를 헤더로 전달하는 흐름을 시각화한다.

### 다음 릴리스 준비 과제
1. Social/Community/Billing 서비스에도 공통 모듈을 적용하고 실제 데이터 저장·조회 로직 연결.
2. Kafka 스트림을 기반으로 한 타임라인 정제 로직, AI 분류기와의 통합, Feature Store 연계를 위한 배치 파이프라인 구현.
3. GraphQL 인증/권한 미들웨어와 Pro 티어 제한, Stripe 웹훅을 통한 `user_entitlements` 업데이트를 연결해 엔드투엔드 유료 기능 흐름 완성.
4. Flyway 마이그레이션 자동화와 GitHub Actions 기반 CI 파이프라인으로 테스트·빌드·배포 체계를 고도화.
