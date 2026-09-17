"""Seed data for RAG chatbot service."""
from __future__ import annotations
from typing import Dict, List

# Sample document collections
SAMPLE_COLLECTIONS: List[dict] = [
    {
        "id": "docs-product-guide",
        "name": "Product Guide",
        "description": "RINCO product documentation and user guides",
        "documents_count": 50,
        "chunk_count": 250,
    },
    {
        "id": "docs-api-reference",
        "name": "API Reference",
        "description": "REST API documentation for all services",
        "documents_count": 30,
        "chunk_count": 180,
    },
    {
        "id": "docs-admin-manual",
        "name": "Admin Manual",
        "description": "System administration and configuration guide",
        "documents_count": 40,
        "chunk_count": 200,
    },
    {
        "id": "docs-faq",
        "name": "FAQ",
        "description": "Frequently asked questions and troubleshooting",
        "documents_count": 25,
        "chunk_count": 100,
    },
]

# Sample Q&A pairs for RAG
SAMPLE_QA_PAIRS: List[dict] = [
    {
        "question": "How do I create a new tenant?",
        "answer": "To create a new tenant, navigate to Admin > Tenants and click 'Add Tenant'. Fill in the required fields: tenant name, domain, and admin email. The tenant will be provisioned with default settings.",
        "category": "administration",
    },
    {
        "question": "What is the difference between hot and warm leads?",
        "answer": "Hot leads have a score above 85% and should be contacted immediately. Warm leads score between 30-85% and benefit from follow-up emails. Cold leads score below 30% and are best suited for nurture campaigns.",
        "category": "lead-scoring",
    },
    {
        "question": "How do I configure webhook notifications?",
        "answer": "Navigate to Settings > Notifications > Webhooks. Click 'Add Webhook' and enter your endpoint URL. Select the events you want to subscribe to (lead.created, deal.stage_changed, etc.) and save.",
        "category": "integrations",
    },
    {
        "question": "Can I export CRM data?",
        "answer": "Yes, go to Data > Export. Select the entity type (leads, deals, contacts), date range, and format (CSV, Excel, JSON). Large exports are processed asynchronously and sent via email.",
        "category": "data-management",
    },
    {
        "question": "How do I set up meeting recording?",
        "answer": "Enable recording in Settings > Meetings > Recording. Choose storage location (MinIO/S3) and quality preset. Recording automatically starts based on your configured rules.",
        "category": "meetings",
    },
]

# Sample knowledge base articles
SAMPLE_ARTICLES: List[dict] = [
    {
        "title": "Getting Started with RINCO CRM",
        "content": "Welcome to RINCO CRM. This guide will help you get started with managing your sales pipeline...",
        "category": "getting-started",
        "tags": ["onboarding", "basics", "setup"],
    },
    {
        "title": "Lead Scoring Best Practices",
        "content": "Lead scoring helps prioritize sales efforts. Key factors include engagement metrics, firmographic data...",
        "category": "lead-management",
        "tags": ["leads", "scoring", "best-practices"],
    },
    {
        "title": "API Authentication Guide",
        "content": "RINCO API uses OAuth 2.0 for authentication. Obtain your client credentials from Settings > API...",
        "category": "api",
        "tags": ["api", "authentication", "oauth"],
    },
    {
        "title": "Meeting Room Setup",
        "content": "Configure meeting rooms with custom layouts, recording settings, and participant permissions...",
        "category": "meetings",
        "tags": ["meetings", "rooms", "configuration"],
    },
]

# Chat context presets
CHAT_CONTEXTS: Dict[str, str] = {
    "sales": "You are a sales assistant helping users with CRM features, lead management, and deal tracking.",
    "technical": "You are a technical support assistant helping with API integration, configuration, and troubleshooting.",
    "admin": "You are an administrative assistant helping with tenant management, user permissions, and system settings.",
    "general": "You are a helpful assistant for RINCO CRM platform. Provide accurate information about features and usage.",
}


def get_sample_collections() -> List[dict]:
    """Return sample collections for seeding."""
    return SAMPLE_COLLECTIONS


def get_sample_qa_pairs() -> List[dict]:
    """Return sample Q&A pairs."""
    return SAMPLE_QA_PAIRS


def get_sample_articles() -> List[dict]:
    """Return sample knowledge base articles."""
    return SAMPLE_ARTICLES


def get_chat_contexts() -> Dict[str, str]:
    """Return chat context presets."""
    return CHAT_CONTEXTS
