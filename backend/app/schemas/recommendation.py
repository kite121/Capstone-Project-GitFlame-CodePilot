from pydantic import BaseModel


class RecommendationCard(BaseModel):
    severity: str
    file: str
    line: int | None = None
    problem: str
    suggestion: str
    confidence: float | None = None


class RecommendationListResponse(BaseModel):
    repository_id: str
    summary: str
    recommendations: list[RecommendationCard]

