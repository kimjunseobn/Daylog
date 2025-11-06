from datetime import datetime
from typing import Any, Dict, List

from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="Daylog Activity Classifier", version="0.1.0")


class ActivityEvent(BaseModel):
    user_id: str
    source: str
    started_at: datetime
    ended_at: datetime
    metadata: Dict[str, Any] = {}


class ClassificationResult(BaseModel):
    category: str
    confidence: float
    rationale: List[str]


class ClassifyResponse(BaseModel):
    status: str
    result: ClassificationResult


@app.get("/healthz")
async def healthcheck():
    return {"status": "ok", "service": "activity-classifier", "time": datetime.utcnow()}


@app.post("/v1/classify", response_model=ClassifyResponse)
async def classify(event: ActivityEvent):
    # TODO: Load ONNX model 및 피처 엔지니어링 적용
    mock_category = "work" if event.source.startswith("notion") else "exercise"
    rationale = [
        f"source={event.source}",
        f"duration_minutes={(event.ended_at - event.started_at).total_seconds() / 60:.1f}",
    ]

    return ClassifyResponse(
        status="ok",
        result=ClassificationResult(
            category=mock_category,
            confidence=0.75,
            rationale=rationale,
        ),
    )
