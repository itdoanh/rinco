"""MinIO storage wrapper."""

from __future__ import annotations

import logging
import os
from typing import Optional

logger = logging.getLogger(__name__)

try:
    from minio import Minio
    from minio.error import S3Error
except Exception:  # pragma: no cover
    Minio = None  # type: ignore
    S3Error = Exception  # type: ignore


class Storage:
    """Thin wrapper around the MinIO client.

    The wrapper lazily connects to MinIO on the first operation. If the
    server is unreachable the layer transparently falls back to stub mode
    so the service can still serve /health and in-memory endpoints in
    development without blocking the import.
    """

    def __init__(
        self,
        host: Optional[str] = None,
        access_key: Optional[str] = None,
        secret_key: Optional[str] = None,
        secure: bool = False,
        bucket: Optional[str] = None,
    ) -> None:
        self.host = host or os.getenv("MINIO_HOST", "localhost:9000")
        self.access_key = access_key or os.getenv("MINIO_USER", "minio")
        self.secret_key = secret_key or os.getenv("MINIO_PASSWORD", "changeme")
        self.secure = secure or os.getenv("MINIO_SECURE", "false").lower() == "true"
        self.bucket = bucket or os.getenv("RECORDINGS_BUCKET", "rinco-recordings")
        self._client = None
        self._ready = False

        # Lazily connect on first use (see _ensure_client).
        if Minio is not None:
            self._client = Minio(
                self.host,
                access_key=self.access_key,
                secret_key=self.secret_key,
                secure=self.secure,
            )

    def _ensure_client(self) -> bool:
        """Probe the connection. Returns True if MinIO is reachable."""
        if self._ready:
            return True
        if self._client is None:
            return False
        try:
            if not self._client.bucket_exists(self.bucket):
                self._client.make_bucket(self.bucket)
            self._ready = True
            return True
        except Exception as exc:  # noqa: BLE001
            logger.debug("MinIO not reachable: %s", exc)
            return False

    @property
    def ready(self) -> bool:
        return self._ensure_client()

    def upload_file(self, source: str, key: str, content_type: str = "video/mp4") -> str:
        if not self._ensure_client():
            return key
        self._client.fput_object(self.bucket, key, source, content_type=content_type)
        return key

    def upload_bytes(self, data: bytes, key: str, content_type: str = "application/json") -> str:
        from io import BytesIO

        if not self._ensure_client():
            return key
        self._client.put_object(
            self.bucket, key, BytesIO(data), length=len(data), content_type=content_type
        )
        return key

    def download_file(self, key: str, target: str) -> None:
        if not self._ensure_client():
            with open(target, "wb") as fp:
                fp.write(b"")
            return
        self._client.fget_object(self.bucket, key, target)

    def remove(self, key: str) -> None:
        if not self._ensure_client():
            return
        try:
            self._client.remove_object(self.bucket, key)
        except Exception as exc:  # noqa: BLE001
            logger.warning("Failed to remove %s: %s", key, exc)

    def presign_get(self, key: str, expires: int = 86400) -> str:
        if not self._ensure_client():
            return f"https://stub/{self.bucket}/{key}?expires={expires}"
        return self._client.presigned_get_object(self.bucket, key, expires=expires)
