from pydantic import BaseModel


class Settings(BaseModel):
    ml_service_url: str = "http://localhost:8001"


settings = Settings()

