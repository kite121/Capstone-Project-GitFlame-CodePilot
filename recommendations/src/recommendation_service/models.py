from enum import StrEnum

from pydantic import BaseModel, ConfigDict, Field, field_validator


class Severity(StrEnum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"


SEVERITY_RANK = {
    Severity.LOW: 0,
    Severity.MEDIUM: 1,
    Severity.HIGH: 2,
}


class Category(StrEnum):
    CODE_DUPLICATION = "code_duplication"
    SECURITY = "security"
    MAINTAINABILITY = "maintainability"
    PERFORMANCE = "performance"
    ARCHITECTURE = "architecture"


ALL_CATEGORIES = list(Category)


class RepoFile(BaseModel):
    model_config = ConfigDict(extra="forbid")

    path: str = Field(min_length=1, max_length=500)
    content: str = Field(max_length=500_000)

    @field_validator("path")
    @classmethod
    def validate_path(cls, value: str) -> str:
        normalized = value.replace("\\", "/").removeprefix("./")
        if normalized.startswith("/") or ".." in normalized.split("/"):
            raise ValueError("path must be a safe repository-relative path")
        return normalized


class AnalyzeRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    config_yaml: str = Field(min_length=1, max_length=100_000)
    repo_context: list[RepoFile] = Field(min_length=1, max_length=2_000)

    @field_validator("repo_context")
    @classmethod
    def paths_must_be_unique(cls, value: list[RepoFile]) -> list[RepoFile]:
        paths = [item.path for item in value]
        if len(paths) != len(set(paths)):
            raise ValueError("repo_context contains duplicate paths")
        return value


class Recommendation(BaseModel):
    model_config = ConfigDict(extra="forbid")

    severity: Severity
    category: Category
    file: str = Field(min_length=1, max_length=500)
    line: int = Field(ge=1)
    problem: str = Field(min_length=1, max_length=800)
    suggestion: str = Field(min_length=1, max_length=800)
    confidence: float = Field(ge=0, le=1)


class RecommendationResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    summary: str = Field(min_length=1, max_length=800)
    recommendations: list[Recommendation] = Field(max_length=10)


class ErrorResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    detail: str


class HealthResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    status: str
    model: str | None = None
