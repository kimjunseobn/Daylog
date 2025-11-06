# Social Feed Service

사용자 맞춤 소셜 피드를 구성하는 베이스라인 API.

## 엔드포인트
- `GET /v1/feed/{userId}`: 사용자의 피드 아이템 리스트 반환

```bash
PORT=7004 POSTGRES_URI=postgres://... go run .
```

- `GET /readyz`로 Postgres/Kafka 연결 상태 확인
- `POST /v1/feed`
  ```json
  {
    "user_id": "uuid",
    "timeline_id": "uuid",
    "category": "work",
    "message": "오늘은 생산적인 하루!",
    "metadata": {"mood": "happy"}
  }
  ```
