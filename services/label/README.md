# Label Service

익명성을 보장하면서도 탐색 가능한 구조화 라벨을 관리한다.

## 엔드포인트
- `GET /v1/labels/{userId}`: 사용자의 라벨 목록 반환
- `POST /v1/labels`: 라벨 생성/수정 (향후 인증 필요)

```bash
PORT=7003 POSTGRES_URI=postgres://... go run .
```

- `GET /readyz`: Postgres 연결 상태 확인
- `POST /v1/labels`
  ```json
  {
    "user_id": "uuid",
    "label_key": "/affiliation",
    "label_value": "서울대학교",
    "is_verified": true,
    "verified_at": "2024-01-01T00:00:00Z"
  }
  ```
