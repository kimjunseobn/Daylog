# Timeline Service

수집된 이벤트를 정제하여 사용자 타임라인을 조회하는 REST API를 제공한다.

## 로컬 실행
```bash
PORT=7000 go run .
```

### 타임라인 조회
```bash
curl http://localhost:7000/v1/timeline/00000000-0000-0000-0000-000000000000
```
