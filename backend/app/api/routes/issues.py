from fastapi import APIRouter

from app.schemas.issue import IssueAnalysisRequest, IssueAnalysisResponse


router = APIRouter(prefix="/issues", tags=["issue workflow"])


@router.post("/analyze", response_model=IssueAnalysisResponse)
def analyze_issue(payload: IssueAnalysisRequest) -> IssueAnalysisResponse:
    return IssueAnalysisResponse(
        status="plan_generated",
        plan_markdown=(
            "# Implementation Plan\n\n"
            "1. Analyze the issue requirements.\n"
            "2. Select relevant repository files.\n"
            "3. Generate implementation steps.\n"
        ),
        notes="Sprint 1 mock response. ML service integration will be added later.",
    )

