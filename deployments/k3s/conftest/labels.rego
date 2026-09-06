package main

# ============================================================
# OPA / conftest — require app.kubernetes.io/* labels
# ============================================================

import rego.v1

required_labels := {"app.kubernetes.io/name", "app.kubernetes.io/instance"}

deny[msg] {
    input.kind == "Deployment"
    some label
    required_labels[label]
    not input.metadata.labels[label]
    msg := sprintf("[labels.rego] Deployment '%s' is missing required label '%s'", [input.metadata.name, label])
}

deny[msg] {
    input.kind == "Deployment"
    c := input.spec.template
    some label
    required_labels[label]
    not c.metadata.labels[label]
    msg := sprintf("[labels.rego] Deployment '%s' pod template is missing label '%s'", [input.metadata.name, label])
}

deny[msg] {
    input.kind == "Service"
    some label
    required_labels[label]
    not input.metadata.labels[label]
    msg := sprintf("[labels.rego] Service '%s' is missing label '%s'", [input.metadata.name, label])
}

# Require name to match naming convention (lowercase, dash-separated)
deny[msg] {
    input.kind == "Deployment"
    not regex.match(`^[a-z][a-z0-9-]{0,62}$`, input.metadata.name)
    msg := sprintf("[labels.rego] Deployment name '%s' violates naming convention", [input.metadata.name])
}
