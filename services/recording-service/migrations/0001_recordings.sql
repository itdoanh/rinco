-- Recording service schema (PostgreSQL).
-- Run via sqlx-cli:  sqlx migrate run
-- Or apply manually:  psql -f migrations/0001_recordings.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS recordings (
    id              UUID         PRIMARY KEY,
    room_id         UUID         NOT NULL,
    tenant_id       UUID         NOT NULL,
    title           TEXT         NOT NULL,
    started_at      TIMESTAMPTZ  NOT NULL,
    ended_at        TIMESTAMPTZ,
    duration_seconds INTEGER     NOT NULL DEFAULT 0,
    size_bytes      BIGINT       NOT NULL DEFAULT 0,
    s3_key          TEXT         NOT NULL,
    s3_bucket       TEXT         NOT NULL,
    tier            TEXT         NOT NULL DEFAULT 'hot',
    format          TEXT         NOT NULL DEFAULT 'mp4',
    status          TEXT         NOT NULL DEFAULT 'pending',
    transcript_url  TEXT,
    thumbnail_url   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_recordings_room    ON recordings (room_id);
CREATE INDEX IF NOT EXISTS idx_recordings_tenant  ON recordings (tenant_id);
CREATE INDEX IF NOT EXISTS idx_recordings_status  ON recordings (status);
CREATE INDEX IF NOT EXISTS idx_recordings_started ON recordings (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_recordings_tier    ON recordings (tier);

CREATE TABLE IF NOT EXISTS transcript_segments (
    id              BIGSERIAL    PRIMARY KEY,
    recording_id    UUID         NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    start_ms        BIGINT       NOT NULL,
    end_ms          BIGINT       NOT NULL,
    text            TEXT         NOT NULL,
    confidence      REAL,
    speaker         TEXT
);

CREATE INDEX IF NOT EXISTS idx_transcript_recording ON transcript_segments (recording_id);
