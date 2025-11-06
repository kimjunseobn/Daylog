# Daylog 초기 구현 범위

아키텍처 명세(`docs/architecture.md`)를 기반으로 한 전체 서비스는 다수의 마이크로서비스, 데이터 파이프라인, ML 워크로드로 구성된다. 본 초기 구현은 향후 기능 확장을 위해 다음과 같은 뼈대를 마련한다.

## 1. 저장소 구조
- `services/`: Go 기반 핵심 마이크로서비스 (수집, 타임라인, 라벨, 소셜, 커뮤니티, 빌링)
- `gateway/`: Node.js(Apollo Server) 기반 GraphQL 게이트웨이
- `ml-services/`: Python(FastAPI) 기반 실시간 추론/추천 서비스
- `jobs/`: 배치 및 ML 파이프라인 스켈레톤
- `infrastructure/`: Terraform/Terragrunt와 Helm 차트 템플릿의 시드 파일
- `db/`: 데이터베이스 스키마 정의 및 마이그레이션 초안
- `docs/`: 문서화 (아키텍처, 구현 범위, 운영 가이드 등)

## 2. 목표 아티팩트
1. 각 서비스별 최소 구동 가능한 API 서버(헬스 체크, 버전 엔드포인트 포함)
2. 공통 환경 구성 (`.env.example`, `Makefile`, `docker-compose.yaml`)으로 로컬 개발 진입장벽 축소
3. GraphQL 게이트웨이에서 서비스 헬스 정보를 federated schema 형태로 노출
4. ML 서비스(활동 분류기) FastAPI 스켈레톤과 학습용 배치 잡 플레이스홀더
5. PostgreSQL 스키마 초안 및 Flyway 마이그레이션 템플릿

## 3. 향후 확장 포인트
- gRPC 인터페이스 추가 및 서비스 간 호출 구현
- Kafka/Flink 기반 스트리밍 파이프라인
- Stripe 연동 및 실제 결제 처리 로직
- 인증/권한 (Cognito/Auth0) 통합
- SageMaker 파이프라인 및 MLflow 연계

본 스켈레톤은 CI/CD, 클라우드 인프라, 세부 비즈니스 로직을 빠르게 확장할 수 있도록 표준 패턴과 모듈 구조를 우선적으로 제공한다.
