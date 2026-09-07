"""Ingestion pipeline: PDF, DOCX, XLSX, MD/TXT, URL."""
from __future__ import annotations

import re
from typing import List, Optional

try:
    import trafilatura  # type: ignore
    HAS_TRAFILATURA = True
except Exception:  # pragma: no cover
    HAS_TRAFILATURA = False
    trafilatura = None  # type: ignore

try:
    import pdfplumber  # type: ignore
    HAS_PDF = True
except Exception:  # pragma: no cover
    HAS_PDF = False
    pdfplumber = None  # type: ignore

try:
    import docx  # type: ignore
    HAS_DOCX = True
except Exception:  # pragma: no cover
    HAS_DOCX = False
    docx = None  # type: ignore

try:
    import openpyxl  # type: ignore
    HAS_XLSX = True
except Exception:  # pragma: no cover
    HAS_XLSX = False
    openpyxl = None  # type: ignore

try:
    import httpx  # type: ignore
    HAS_HTTPX = True
except Exception:  # pragma: no cover
    HAS_HTTPX = False
    httpx = None  # type: ignore

from app.core import get_logger

log = get_logger("rag-chatbot.ingestion")


async def fetch_url(url: str) -> str:
    """Extract text from a URL using trafilatura."""
    if HAS_TRAFILATURA:
        try:
            downloaded = trafilatura.fetch_url(url)
            if downloaded:
                text = trafilatura.extract(downloaded)
                if text:
                    return text
        except Exception as exc:
            log.warning("trafilatura_failed", url=url, error=str(exc))
    if HAS_HTTPX:
        try:
            async with httpx.AsyncClient(timeout=15, follow_redirects=True) as client:
                r = await client.get(url)
                if r.status_code >= 300:
                    raise RuntimeError(f"HTTP {r.status_code}")
                html = r.text
            text = re.sub(r"<[^>]+>", " ", html)
            return re.sub(r"\s+", " ", text).strip()
        except Exception as exc:
            log.warning("httpx_fetch_failed", url=url, error=str(exc))
    return ""


def extract_pdf(text: str) -> str:
    """Extract text from a PDF using pdfplumber."""
    if not HAS_PDF:
        return text  # return as-is when pdfplumber unavailable
    try:
        import io
        with pdfplumber.open(io.BytesIO(text.encode())) as pdf:
            pages = [p.extract_text() or "" for p in pdf.pages]
            return "\n\n".join(pages)
    except Exception as exc:
        log.warning("pdfplumber_failed", error=str(exc))
        return text


def extract_docx(text: str) -> str:
    """Extract text from a DOCX file."""
    if not HAS_DOCX:
        return text
    try:
        import io
        doc = docx.Document(io.BytesIO(text.encode()))
        return "\n\n".join(p.text for p in doc.paragraphs)
    except Exception as exc:
        log.warning("docx_failed", error=str(exc))
        return text


def extract_xlsx(text: str) -> str:
    """Extract text from an XLSX file as CSV-like rows."""
    if not HAS_XLSX:
        return text
    try:
        import io
        wb = openpyxl.load_workbook(io.BytesIO(text.encode()))
        lines: List[str] = []
        for sheet in wb.sheetnames:
            ws = wb[sheet]
            lines.append(f"[Sheet: {sheet}]")
            for row in ws.iter_rows(values_only=True):
                row_str = " | ".join(str(c or "") for c in row)
                if row_str.strip():
                    lines.append(row_str)
        return "\n".join(lines)
    except Exception as exc:
        log.warning("xlsx_failed", error=str(exc))
        return text


def extract_kind(kind: str, content: str) -> str:
    """Dispatch to the right extractor based on content kind."""
    if kind == "pdf":
        return extract_pdf(content)
    if kind == "docx":
        return extract_docx(content)
    if kind == "xlsx":
        return extract_xlsx(content)
    return content  # text, md, txt — return as-is


def ingest_content(source: str, kind: str) -> str:
    """High-level: fetch URL or return raw content as text."""
    if kind == "url":
        import asyncio
        return asyncio.get_event_loop().run_until_complete(fetch_url(source))
    return source  # raw text / already-parsed content