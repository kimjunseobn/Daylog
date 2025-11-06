# Daylog Monorepo Skeleton

`docs/architecture.md` 명세를 바탕으로 Daylog 서비스를 구현하기 위한 초기 코드 베이스입니다.  
여러 마이크로서비스, ML 컴포넌트, 인프라 구성을 모노레포 하나에서 관리할 수 있도록 스켈레톤을 제공합니다.

## 구성 개요
- `gateway/`: GraphQL BFF(Apollo Federation) 게이트웨이
- `services/`: Go 기반 도메인 마이크로서비스 묶음
- `ml-services/`: Python FastAPI 기반 실시간 추론·추천 서비스
- `jobs/`: 배치 작업 및 학습 파이프라인 템플릿
- `db/`: 데이터베이스 스키마 및 마이그레이션
- `infrastructure/`: Terraform/Helm 등 IaC 아티팩트
- `docs/`: 설계 및 운영 문서
- `app/`: React Native(Expo) 기반 모바일 앱

## 빠른 시작
```bash
# 환경 변수 준비
cp .env.example .env

# Go/NPM/Python 의존성 설치
make bootstrap

# 로컬 서비스 기동 (Docker Compose 사용 예정)
make dev
```

<<<<<<< HEAD
각 서비스에 대한 자세한 실행 방법은 하위 디렉터리의 README를 참고하세요.

### 모바일 앱 실행
=======
각 서비스에 대한 자세한 실행 방법은 하위 디렉토리의 README를 참고하세요.

모바일 앱 실행:
>>>>>>> 9dd3b40 (2)
```bash
cd app
npm install
EXPO_PUBLIC_GATEWAY_URL=http://localhost:4000/graphql npm run start
```
<<<<<<< HEAD

> 에뮬레이터에서는 게이트웨이 주소를 `http://10.0.2.2:4000/graphql`(Android) 또는 `http://127.0.0.1:4000/graphql`(iOS)로 변경해 주세요.
=======
>>>>>>> 9dd3b40 (2)
