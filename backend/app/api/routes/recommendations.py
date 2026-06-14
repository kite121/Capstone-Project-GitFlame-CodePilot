from fastapi import APIRouter

from app.schemas.recommendation import RecommendationListResponse


router = APIRouter(tags=["recommendations"])


@router.get("/{repository_id}/recommendations", response_model=RecommendationListResponse)
def list_recommendations(repository_id: str) -> RecommendationListResponse:
    return RecommendationListResponse(
        repository_id=repository_id,
        summary="Sprint 1 mock recommendation summary.",
        recommendations=[],
    )

