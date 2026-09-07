"""Async clients to PG/CH/Mongo/NATS.

The clients are lazy: they connect on first use and gracefully tolerate
unreachable infrastructure so the service can boot in dev without any
backend.
"""
from __future__ import annotations

import os
from typing import Any, Optional


class LazyClient:
    """Base class for clients that connect on first use."""

    def __init__(self, url: str = "") -> None:
        self.url = url
        self._client: Optional[Any] = None

    @property
    def client(self) -> Any:
        if self._client is None and self.url:
            self._client = self._connect()
        return self._client

    def _connect(self) -> Any:  # pragma: no cover - real impl in subclass
        raise NotImplementedError

    async def aclose(self) -> None:
        if self._client is None:
            return
        try:
            if hasattr(self._client, "close"):
                result = self._client.close()
                if hasattr(result, "__await__"):
                    await result
            elif hasattr(self._client, "aclose"):
                await self._client.aclose()
        except Exception:
            pass
        self._client = None


class PostgresClient(LazyClient):
    """asyncpg pool wrapper."""

    def _connect(self) -> Any:  # pragma: no cover
        try:
            import asyncpg  # type: ignore

            return asyncpg
        except Exception:
            return None

    async def fetch(self, query: str, *args: Any) -> list:
        try:
            import asyncpg  # type: ignore

            if self._client is None or not isinstance(self._client, asyncpg.Pool):
                if self.url:
                    self._client = await asyncpg.create_pool(self.url)
            if not self._client:
                return []
            async with self._client.acquire() as conn:
                return await conn.fetch(query, *args)
        except Exception:
            return []


class ClickHouseClient(LazyClient):
    """clickhouse-driver async wrapper."""

    def _connect(self) -> Any:  # pragma: no cover
        try:
            import clickhouse_driver  # type: ignore

            return clickhouse_driver
        except Exception:
            return None

    async def execute(self, query: str, *args: Any) -> Any:
        try:
            if not self.url:
                return None
            from clickhouse_driver import Client  # type: ignore

            client = Client.from_url(self.url) if hasattr(Client, "from_url") else Client(self.url)
            return client.execute(query, args)
        except Exception:
            return None


class MongoClient(LazyClient):
    """motor (Mongo async) wrapper."""

    def _connect(self) -> Any:  # pragma: no cover
        try:
            import motor.motor_asyncio as motor  # type: ignore

            return motor
        except Exception:
            return None

    def db(self) -> Any:
        if not self.url:
            return None
        try:
            import motor.motor_asyncio as motor  # type: ignore

            if self._client is None:
                self._client = motor.AsyncIOMotorClient(self.url)
            return self._client
        except Exception:
            return None


class NATSClient(LazyClient):
    """nats-py wrapper."""

    def _connect(self) -> Any:  # pragma: no cover
        try:
            import nats  # type: ignore

            return nats
        except Exception:
            return None


def postgres() -> PostgresClient:
    return PostgresClient(os.getenv("LEAD_SCORING_POSTGRES_URL", os.getenv("DATABASE_URL", "")))


def clickhouse() -> ClickHouseClient:
    return ClickHouseClient(os.getenv("LEAD_SCORING_CLICKHOUSE_URL", os.getenv("CLICKHOUSE_URL", "")))


def mongo() -> MongoClient:
    return MongoClient(os.getenv("LEAD_SCORING_MONGO_URL", os.getenv("MONGO_URL", "")))


def nats() -> NATSClient:
    return NATSClient(os.getenv("LEAD_SCORING_NATS_URL", os.getenv("NATS_URL", "")))


__all__ = [
    "LazyClient",
    "PostgresClient",
    "ClickHouseClient",
    "MongoClient",
    "NATSClient",
    "postgres",
    "clickhouse",
    "mongo",
    "nats",
]