"""GitHub client for creating hotfix PRs."""
from __future__ import annotations

import base64
import os
from typing import Any

import httpx

from app.core import get_logger

GITHUB_TOKEN = os.getenv("GITHUB_TOKEN", "")
GITHUB_REPO = os.getenv("GITHUB_REPO", "itdoanh/rinco")
GITHUB_API = "https://api.github.com"

logger = get_logger("github_client")

# Optional PyGithub wrapper
try:
    from github import Github as PyGithub  # type: ignore
    _HAS_PYGITHUB = True
except ImportError:
    _HAS_PYGITHUB = False
    PyGithub = None


async def get_file_content(repo: str, path: str, ref: str = "main") -> str:
    """
    Fetch the content of a file from a GitHub repository at a given ref.

    Args:
        repo: Repository in "owner/repo" format.
        path: Path to the file within the repository.
        ref: Git branch, tag, or commit SHA. Defaults to "main".

    Returns:
        The decoded file content as a string, or empty string on failure.
    """
    if not GITHUB_TOKEN:
        logger.warning("github_token_missing")
        return ""

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            r = await client.get(
                f"{GITHUB_API}/repos/{repo}/contents/{path}",
                params={"ref": ref},
                headers={
                    "Authorization": f"Bearer {GITHUB_TOKEN}",
                    "Accept": "application/vnd.github.v3+json",
                    "X-GitHub-Api-Version": "2022-11-28",
                },
            )
            r.raise_for_status()
            data = r.json()
            return base64.b64decode(data["content"]).decode("utf-8", errors="ignore")
    except httpx.HTTPStatusError as e:
        logger.warning("github_file_not_found", repo=repo, path=path, status=e.response.status_code)
        return ""
    except Exception as e:
        logger.warning("github_file_fetch_failed", repo=repo, path=path, error=str(e))
        return ""


async def create_pr(
    title: str,
    body: str,
    branch: str,
    base: str,
    file_path: str,
    new_content: str,
    sha: str,
    repo: str | None = None,
) -> str:
    """
    Create a pull request with a file change on a new branch.

    Args:
        title: PR title.
        body: PR description.
        branch: Feature branch name to create.
        base: Target branch (e.g. "main").
        file_path: Path to the file being changed.
        new_content: New file content (full file, not just diff).
        sha: SHA of the current file (for update detection).
        repo: Repository in "owner/repo" format. Defaults to GITHUB_REPO env var.

    Returns:
        URL of the created PR, or empty string on failure.
    """
    repo = repo or GITHUB_REPO
    if not GITHUB_TOKEN:
        logger.warning("github_token_missing")
        return ""

    try:
        async with httpx.AsyncClient(timeout=30.0) as client:
            auth_headers = {
                "Authorization": f"Bearer {GITHUB_TOKEN}",
                "Accept": "application/vnd.github.v3+json",
                "X-GitHub-Api-Version": "2022-11-28",
            }

            # 1. Create branch from main's HEAD SHA
            ref_r = await client.get(
                f"{GITHUB_API}/repos/{repo}/git/refs/heads/{base}",
                headers=auth_headers,
            )
            ref_r.raise_for_status()
            base_sha = ref_r.json()["object"]["sha"]

            await client.post(
                f"{GITHUB_API}/repos/{repo}/git/refs",
                headers=auth_headers,
                json={"ref": f"refs/heads/{branch}", "sha": base_sha},
            )

            # 2. Update (or create) the file
            await client.put(
                f"{GITHUB_API}/repos/{repo}/contents/{file_path}",
                headers=auth_headers,
                json={
                    "message": title,
                    "branch": branch,
                    "content": base64.b64encode(new_content.encode()).decode(),
                    "sha": sha,
                },
            )

            # 3. Create PR
            pr_r = await client.post(
                f"{GITHUB_API}/repos/{repo}/pulls",
                headers=auth_headers,
                json={
                    "title": title,
                    "head": branch,
                    "base": base,
                    "body": body,
                },
            )
            pr_r.raise_for_status()
            pr_url = pr_r.json().get("html_url", "")
            logger.info("github_pr_created", pr_url=pr_url)
            return pr_url

    except httpx.HTTPStatusError as e:
        logger.warning("github_pr_failed", status=e.response.status_code, error=str(e))
        return ""
    except Exception as e:
        logger.warning("github_pr_error", error=str(e))
        return ""


async def create_pr_with_pygithub(
    title: str,
    body: str,
    branch: str,
    base: str,
    file_path: str,
    new_content: str,
    sha: str,
    repo: str | None = None,
) -> str:
    """
    Create a PR using PyGithub (falls back to httpx-based version if unavailable).
    """
    if not _HAS_PYGITHUB:
        return await create_pr(title, body, branch, base, file_path, new_content, sha, repo)

    repo = repo or GITHUB_REPO
    try:
        client = PyGithub(GITHUB_TOKEN)
        gh_repo = client.get_repo(repo)
        main_ref = gh_repo.get_git_ref(f"heads/{base}")
        gh_repo.create_git_ref(ref=f"refs/heads/{branch}", sha=main_ref.object.sha)
        contents = gh_repo.get_contents(file_path, ref=base)
        gh_repo.update_file(
            path=file_path,
            message=title,
            content=new_content,
            sha=contents.sha,
            branch=branch,
        )
        pr = gh_repo.create_pull(title=title, body=body, head=branch, base=base)
        return pr.html_url
    except Exception as e:
        logger.warning("pygithub_pr_failed", error=str(e))
        return await create_pr(title, body, branch, base, file_path, new_content, sha, repo)
