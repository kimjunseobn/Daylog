# Label Service

익명성을 보장하면서도 탐색 가능한 구조화 라벨을 관리한다.

## 엔드포인트
- `GET /v1/labels/{userId}`: 사용자의 라벨 목록 반환
- `POST /v1/labels`: 라벨 생성/수정 (향후 인증 필요)

```bash
PORT=7000 go run .
```
