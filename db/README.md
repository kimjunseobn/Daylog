# 데이터베이스 스키마

- `schema.sql`: 핵심 테이블 요약본
- `migrations/`: Flyway 호환 마이그레이션 파일

로컬 개발:
```bash
docker compose up -d postgres
psql postgres://daylog:daylog@localhost:5432/daylog -f db/schema.sql
```
