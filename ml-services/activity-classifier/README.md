# Activity Classifier Service

FastAPI 기반의 실시간 활동 분류 엔드포인트.

## 실행
```bash
python -m venv .venv
.venv/Scripts/activate  # Windows
pip install -r requirements.txt
uvicorn app.main:app --reload
```

### 분류 요청 예시
```bash
curl -X POST http://localhost:8000/v1/classify \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "00000000-0000-0000-0000-000000000000",
    "source": "notion.desktop",
    "started_at": "2024-01-01T02:00:00Z",
    "ended_at": "2024-01-01T03:00:00Z",
    "metadata": {"window_title": "Project Spec"}
  }'
```
