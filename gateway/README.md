# Daylog GraphQL Gateway

Apollo Server 기반의 단일 GraphQL 엔드포인트로 각 마이크로서비스를 집계한다.

## 로컬 실행
```bash
cd gateway
npm install
GATEWAY_PORT=4000 \
TIMELINE_SERVICE_URL=http://localhost:7002 \
LABEL_SERVICE_URL=http://localhost:7003 \
SOCIAL_FEED_SERVICE_URL=http://localhost:7004 \
COMMUNITY_SERVICE_URL=http://localhost:7005 \
BILLING_SERVICE_URL=http://localhost:7006 \
npm run dev
```

GraphQL 플레이그라운드: <http://localhost:4000/graphql>

### 예시 쿼리
```graphql
query Example {
  timeline(userId: "00000000-0000-0000-0000-000000000000", limit: 10) {
    event_id
    category
    metadata
  }
}
```

라벨 생성/수정:
```graphql
mutation {
  upsertLabel(
    input: {
      user_id: "00000000-0000-0000-0000-000000000000"
      label_key: "/affiliation"
      label_value: "서울대학교"
      is_verified: true
    }
  ) {
    id
    label_value
    is_verified
  }
}
```

커뮤니티 생성:
```graphql
mutation {
  createCommunity(input: { title: "아침 루틴", description: "기상 공유", is_pro_only: false }) {
    id
    title
    created_at
  }
}
```

현재는 간단한 헤더 기반 인증을 사용합니다.

- `x-user-id`: 현재 사용자 ID
- `x-user-tier`: `free` | `pro`

예)
```bash
curl http://localhost:4000/graphql \
  -H "x-user-id: 00000000-0000-0000-0000-000000000000" \
  -H "x-user-tier: pro" \
  -H "Content-Type: application/json" \
  -d '{"query":"{ viewerEntitlement { tier status } }"}'
```
