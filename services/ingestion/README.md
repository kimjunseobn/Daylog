# Ingestion Service

사용자의 기기/앱에서 수집된 이벤트를 수신하고 Kafka로 퍼블리시하는 엔드포인트를 제공한다.

## 로컬 실행
```bash
PORT=7000 go run .
```

### 헬스 체크
```bash
curl http://localhost:7000/healthz
```

### 이벤트 수집 예시
```bash
curl -X POST http://localhost:7000/v1/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "00000000-0000-0000-0000-000000000000",
    "source": "ios_screen_time",
    "started_at": "2024-01-01T09:00:00Z",
    "ended_at": "2024-01-01T10:00:00Z",
    "metadata": {"bundle_id": "com.daylog"}
  }'
```
