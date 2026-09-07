"""Uvicorn entrypoint.

Run::

    uvicorn main:app --host 0.0.0.0 --port 8092

The app is defined in `app.main`; this file simply re-exports it so
`uvicorn main:app` works out of the box.
"""
from __future__ import annotations

from app.main import app, create_app

__all__ = ["app", "create_app"]


if __name__ == "__main__":  # pragma: no cover
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8092)