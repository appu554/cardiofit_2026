import os
from pydantic_settings import BaseSettings
from typing import List
from dotenv import load_dotenv

load_dotenv()

class Settings(BaseSettings):
    # Supabase Configuration
    SUPABASE_URL: str = os.getenv("SUPABASE_URL", "https://auugxeqzgrnknklgwqrh.supabase.co")
    SUPABASE_KEY: str = os.getenv("SUPABASE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8")
    SUPABASE_JWT_SECRET: str = os.getenv("SUPABASE_JWT_SECRET", "")  # This should be set in production
    SUPABASE_ALGORITHMS: List[str] = ["HS256"]  # Supabase uses HS256 by default

    # Legacy Auth0 Configuration (kept for backward compatibility)
    AUTH0_DOMAIN: str = os.getenv("AUTH0_DOMAIN", "")
    AUTH0_API_AUDIENCE: str = os.getenv("AUTH0_API_AUDIENCE", "")
    AUTH0_ISSUER: str = f"https://{AUTH0_DOMAIN}/"
    AUTH0_ALGORITHMS: List[str] = ["RS256"]
    AUTH0_CLIENT_ID: str = os.getenv("AUTH0_CLIENT_ID", "")
    AUTH0_CLIENT_SECRET: str = os.getenv("AUTH0_CLIENT_SECRET", "")
    AUTH0_MGMT_CLIENT_ID: str = os.getenv("AUTH0_MGMT_CLIENT_ID", "")
    AUTH0_MGMT_CLIENT_SECRET: str = os.getenv("AUTH0_MGMT_CLIENT_SECRET", "")
    AUTH0_MGMT_AUDIENCE: str = f"https://{AUTH0_DOMAIN}/api/v2/"

    # API Configuration
    API_PREFIX: str = "/api"
    PROJECT_NAME: str = "Auth Service"
    DEBUG: bool = os.getenv("DEBUG", "true").lower() == "true"  # Set to true by default for development
    FORCE_AUTH_SUCCESS: bool = os.getenv("FORCE_AUTH_SUCCESS", "true").lower() == "true"  # Set to true by default for development

settings = Settings()