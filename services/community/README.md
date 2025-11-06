# Community Service

커뮤니티, 챌린지, 멤버십을 관리하는 API 스켈레톤.

## 사용법
```bash
PORT=7005 POSTGRES_URI=postgres://... go run .
```

- `GET /readyz`: DB 연결 체크
- `POST /v1/communities` 및 `POST /v1/communities/{id}/join` 사용 예시는 API 스펙 참고
