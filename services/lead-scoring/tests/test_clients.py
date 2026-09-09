"""Extra tests for lead-scoring clients."""
import os
import pytest

from app.services.clients import (
    LazyClient,
    PostgresClient,
    ClickHouseClient,
    MongoClient,
    NATSClient,
    postgres,
    clickhouse,
    mongo,
    nats,
)


def test_lazy_client_default_url():
    c = LazyClient()
    assert c.url == ""
    assert c._client is None


def test_lazy_client_provided_url():
    c = LazyClient("postgres://x")
    assert c.url == "postgres://x"


def test_lazy_client_no_url_no_client():
    c = LazyClient()
    assert c.client is None


def test_lazy_client_aclose_no_client():
    import asyncio

    async def run():
        c = LazyClient()
        await c.aclose()
        assert c._client is None

    asyncio.run(run())


def test_lazy_client_aclose_with_close_method():
    import asyncio

    closed = {"value": False}

    class FakeClient:
        def close(self):
            closed["value"] = True

    async def run():
        c = LazyClient()
        c._client = FakeClient()
        await c.aclose()
        assert closed["value"] is True

    asyncio.run(run())


def test_lazy_client_aclose_with_aclose():
    import asyncio

    closed = {"value": False}

    class FakeClient:
        async def aclose(self):
            closed["value"] = True

    async def run():
        c = LazyClient()
        c._client = FakeClient()
        await c.aclose()
        assert closed["value"] is True

    asyncio.run(run())


def test_lazy_client_aclose_with_no_methods():
    import asyncio

    class FakeClient:
        pass

    async def run():
        c = LazyClient()
        c._client = FakeClient()
        await c.aclose()
        assert c._client is None

    asyncio.run(run())


def test_lazy_client_aclose_exception_safe():
    import asyncio

    class FakeClient:
        def close(self):
            raise RuntimeError("boom")

    async def run():
        c = LazyClient()
        c._client = FakeClient()
        # Should not raise
        await c.aclose()

    asyncio.run(run())


def test_postgres_client_default():
    c = PostgresClient()
    assert isinstance(c, LazyClient)


def test_postgres_client_provided_url():
    c = PostgresClient("postgres://test")
    assert c.url == "postgres://test"


def test_postgres_fetch_no_url():
    import asyncio

    async def run():
        c = PostgresClient()
        result = await c.fetch("SELECT 1")
        assert result == []

    asyncio.run(run())


def test_clickhouse_client_default():
    c = ClickHouseClient()
    assert isinstance(c, LazyClient)


def test_clickhouse_execute_no_url():
    import asyncio

    async def run():
        c = ClickHouseClient()
        result = await c.execute("SELECT 1")
        assert result is None

    asyncio.run(run())


def test_mongo_client_default():
    c = MongoClient()
    assert isinstance(c, LazyClient)


def test_mongo_db_no_url():
    c = MongoClient()
    assert c.db() is None


def test_nats_client_default():
    c = NATSClient()
    assert isinstance(c, LazyClient)


def test_postgres_factory_default_url(monkeypatch):
    monkeypatch.delenv("LEAD_SCORING_POSTGRES_URL", raising=False)
    monkeypatch.delenv("DATABASE_URL", raising=False)
    c = postgres()
    assert c.url == ""


def test_postgres_factory_with_env(monkeypatch):
    monkeypatch.setenv("DATABASE_URL", "postgres://test")
    c = postgres()
    assert c.url == "postgres://test"


def test_clickhouse_factory_default_url(monkeypatch):
    monkeypatch.delenv("LEAD_SCORING_CLICKHOUSE_URL", raising=False)
    monkeypatch.delenv("CLICKHOUSE_URL", raising=False)
    c = clickhouse()
    assert c.url == ""


def test_clickhouse_factory_with_env(monkeypatch):
    monkeypatch.setenv("CLICKHOUSE_URL", "clickhouse://test")
    c = clickhouse()
    assert c.url == "clickhouse://test"


def test_mongo_factory_default_url(monkeypatch):
    monkeypatch.delenv("LEAD_SCORING_MONGO_URL", raising=False)
    monkeypatch.delenv("MONGO_URL", raising=False)
    c = mongo()
    assert c.url == ""


def test_mongo_factory_with_env(monkeypatch):
    monkeypatch.setenv("MONGO_URL", "mongodb://test")
    c = mongo()
    assert c.url == "mongodb://test"


def test_nats_factory_default_url(monkeypatch):
    monkeypatch.delenv("LEAD_SCORING_NATS_URL", raising=False)
    monkeypatch.delenv("NATS_URL", raising=False)
    c = nats()
    assert c.url == ""


def test_nats_factory_with_env(monkeypatch):
    monkeypatch.setenv("NATS_URL", "nats://test")
    c = nats()
    assert c.url == "nats://test"


def test_postgres_factory_lead_scoring_env(monkeypatch):
    monkeypatch.setenv("LEAD_SCORING_POSTGRES_URL", "postgres://scoring")
    c = postgres()
    assert c.url == "postgres://scoring"
