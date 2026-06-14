from fastapi import FastAPI

from app.api.routes import health, issues, recommendations


app = FastAPI(
    title="GitFlame CodePilot Backend",
    description="External AI integration service for GitFlame workflows.",
    version="0.1.0",
)

app.include_router(health.router)
app.include_router(issues.router, prefix="/integrations/gitflame")
app.include_router(recommendations.router, prefix="/repositories")

