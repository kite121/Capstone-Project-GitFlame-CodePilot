from fastapi import FastAPI

from app.schemas import IssuePlanRequest, IssuePlanResponse, RecommendationRequest, RecommendationResponse


app = FastAPI(
    title="GitFlame CodePilot ML Service",
    description="Mock ML service for issue planning, code generation, and recommendations.",
    version="0.1.0",
)


@app.get("/health")
def health_check() -> dict[str, str]:
    return {"status": "ok", "service": "ml_service"}


@app.post("/issue-plan", response_model=IssuePlanResponse)
def generate_issue_plan(payload: IssuePlanRequest) -> IssuePlanResponse:
    return IssuePlanResponse(
        plan_markdown=(
            "# Implementation Plan\n\n"
            "1. Understand the issue.\n"
            "2. Retrieve relevant repository context.\n"
            "3. Generate implementation steps.\n"
        )
    )


@app.post("/recommendations", response_model=RecommendationResponse)
def generate_recommendations(payload: RecommendationRequest) -> RecommendationResponse:
    return RecommendationResponse(
        summary=(
            "Repository analysis found 3 improvement opportunities: one security item, "
            "one maintainability item, and one performance item. No blocking defects were detected."
        ),
        recommendations=[
            {
                "severity": "high",
                "file": "backend/internal/app/server.go",
                "line": 142,
                "problem": "Repository identifiers should be validated before they are used in API routes and logs.",
                "suggestion": "Validate repository ids at the request boundary and keep structured fields separate from log messages.",
                "confidence": 0.86,
            },
            {
                "severity": "medium",
                "file": "backend/internal/app/storage.go",
                "line": 88,
                "problem": "Recommendation state is stored only in memory, so data is lost after a backend restart.",
                "suggestion": "Connect the recommendation flow to the PostgreSQL tables from backend/db/schema.sql.",
                "confidence": 0.78,
            },
            {
                "severity": "low",
                "file": "README.md",
                "line": 34,
                "problem": "The run instructions are correct, but troubleshooting notes are still minimal.",
                "suggestion": "Add a short troubleshooting section for occupied ports and Docker image pull failures.",
                "confidence": 0.64,
            },
        ],
    )

