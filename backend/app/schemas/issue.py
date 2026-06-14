from pydantic import BaseModel, Field


class IssueAnalysisRequest(BaseModel):
    repository_id: str
    issue_title: str
    issue_body: str
    yaml_config: str = Field(description="Repository AI configuration content.")


class IssueAnalysisResponse(BaseModel):
    status: str
    plan_markdown: str
    notes: str | None = None

