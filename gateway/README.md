# Daylog GraphQL Gateway

Apollo Server 기반의 단일 GraphQL 엔드포인트로 각 마이크로서비스를 집계한다.

## 로컬 실행
```bash
cd gateway
npm install
npm run dev
```

GraphQL 플레이그라운드: <http://localhost:4000/graphql>

### 예시 쿼리
```graphql
query Example {
  timeline(userId: "00000000-0000-0000-0000-000000000000") {
    category
    started_at
  }
}
```
