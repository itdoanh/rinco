"""Pydantic-settings driven configuration."""
from __future__ import annotations

import os
from functools import lru_cache
from typing import Any

from pydantic import Field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Lead scoring service configuration."""

    model_config = SettingsConfigDict(
        env_prefix="LEAD_SCORING_",
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    service_name: str = "lead-scoring"
    version: str = "2.0.0"
    env: str = Field(default="development")
    http_addr: str = ":8092"
    log_level: str = "INFO"

    # Model
    model_dir: str = "/models"
    fallback_model_path: str = "/models/fallback/logreg.pkl"
    default_model_version: str = "1.0.0"

    # Feature store / clients
    postgres_url: str = ""
    clickhouse_url: str = ""
    mongo_url: str = ""
    nats_url: str = ""
    nats_subject: str = "lead.created"

    # MLflow
    mlflow_tracking_uri: str = ""
    mlflow_experiment: str = "lead-scoring"

    # Limits
    max_batch_size: int = 1000
    allow_public_train: bool = False


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    """Return cached Settings instance."""
    return Settings()


def env_or(key: str, default: Any) -> Any:
    return os.getenv(key, default)