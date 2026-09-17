"""Seed data for AI SRE service."""
from __future__ import annotations
from typing import Dict, List

# Sample incidents for testing
SAMPLE_INCIDENTS: List[dict] = [
    {
        "service": "payment-service",
        "severity": "P0",
        "error": "Connection timeout to payment gateway",
        "stack": "TimeoutError: Connection refused at payment/gateway.py:45",
        "trace_id": "abc123def456",
        "status": "open",
    },
    {
        "service": "user-service",
        "severity": "P1",
        "error": "Database connection pool exhausted",
        "stack": "PoolExhaustedError: All connections in use at db/pool.py:120",
        "status": "investigating",
    },
    {
        "service": "notification-service",
        "severity": "P2",
        "error": "Email delivery delay exceeding SLA",
        "stack": "SMTPTimeout: Connection timed out after 30s",
        "status": "open",
    },
    {
        "service": "auth-service",
        "severity": "P1",
        "error": "JWT token validation failure",
        "stack": "JWTError: Signature verification failed at auth/jwt.py:89",
        "trace_id": "jwt789xyz123",
        "status": "resolved",
    },
    {
        "service": "analytics-service",
        "severity": "P2",
        "error": "Query timeout on large dataset",
        "stack": "QueryTimeout: Execution time exceeded 60s at analytics/query.py:200",
        "status": "open",
    },
]

# Common error patterns
ERROR_PATTERNS: Dict[str, dict] = {
    "timeout": {
        "category": "Network",
        "likely_causes": ["Service unavailable", "High latency", "Network partition"],
        "recommended_actions": ["Check service health", "Review timeout configs", "Scale horizontally"],
    },
    "connection_refused": {
        "category": "Network",
        "likely_causes": ["Service down", "Port blocked", "Firewall rules"],
        "recommended_actions": ["Verify service status", "Check port accessibility", "Review firewall"],
    },
    "database_error": {
        "category": "Database",
        "likely_causes": ["Query inefficiency", "Connection pool exhaustion", "Deadlock"],
        "recommended_actions": ["Check query performance", "Review connection pool settings", "Analyze locks"],
    },
    "authentication_error": {
        "category": "Security",
        "likely_causes": ["Invalid credentials", "Expired tokens", "Key rotation"],
        "recommended_actions": ["Verify credentials", "Check token expiry", "Review key configuration"],
    },
    "memory_error": {
        "category": "Infrastructure",
        "likely_causes": ["Memory leak", "Insufficient resources", "Large data processing"],
        "recommended_actions": ["Profile memory usage", "Review resource limits", "Optimize data handling"],
    },
}

# Runbook templates
RUNBOOK_TEMPLATES: Dict[str, str] = {
    "database_incident": """# Database Incident Runbook

## Symptoms
- Connection timeouts
- Slow queries
- Database errors

## Diagnosis Steps
1. Check database server health
2. Review active connections
3. Analyze slow queries
4. Check disk space

## Resolution Steps
1. Kill long-running queries if necessary
2. Scale up connection pool
3. Consider read replicas
4. Optimize problematic queries

## Prevention
- Regular index maintenance
- Connection pool tuning
- Query performance monitoring
""",
    "network_incident": """# Network Incident Runbook

## Symptoms
- Connection timeouts
- Service unreachable
- High latency

## Diagnosis Steps
1. Check network connectivity
2. Review firewall rules
3. Analyze network metrics
4. Check DNS resolution

## Resolution Steps
1. Restart affected services
2. Update firewall rules if needed
3. Scale services to healthy nodes
4. Update DNS if necessary

## Prevention
- Health check monitoring
- Circuit breaker patterns
- Graceful degradation
""",
    "auth_incident": """# Authentication Incident Runbook

## Symptoms
- Login failures
- Token validation errors
- Permission denied errors

## Diagnosis Steps
1. Check authentication service logs
2. Verify token signing keys
3. Review user permissions
4. Check token expiration settings

## Resolution Steps
1. Rotate signing keys if compromised
2. Clear token cache
3. Force re-authentication
4. Update token expiration settings

## Prevention
- Regular key rotation
- Token expiration policies
- Audit logging
""",
}


def get_sample_incidents() -> List[dict]:
    """Return sample incidents for testing."""
    return SAMPLE_INCIDENTS


def get_error_patterns() -> Dict[str, dict]:
    """Return error pattern mappings."""
    return ERROR_PATTERNS


def get_runbook_templates() -> Dict[str, str]:
    """Return runbook templates."""
    return RUNBOOK_TEMPLATES
