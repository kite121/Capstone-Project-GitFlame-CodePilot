from pydantic import BaseModel


class IssuePlanRequest(BaseModel):
    issue_title: str
    issue_body: str
    yaml_config: str
    repository_context: list[str] = []


class IssuePlanResponse(BaseModel):
    plan_markdown: str


class RecommendationRequest(BaseModel):
    yaml_config: str
    repository_context: list[str] = []


class RecommendationCard(BaseModel):
    severity: str
    file: str
    line: int | None = None
    problem: str
    suggestion: str
    confidence: float | None = None


class RecommendationResponse(BaseModel):
    summary: str
    recommendations: list[RecommendationCard]

