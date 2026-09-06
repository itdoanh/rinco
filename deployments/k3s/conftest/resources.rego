package main

# ============================================================
# OPA / conftest — require resource limits & requests
# ============================================================

import rego.v1

deny[msg] {
    input.kind == "Deployment"
    some i
    c := input.spec.template.spec.containers[i]
    not c.resources.limits.cpu
    msg := sprintf("[resources.rego] container '%s' is missing resources.limits.cpu", [c.name])
}

deny[msg] {
    input.kind == "Deployment"
    some i
    c := input.spec.template.spec.containers[i]
    not c.resources.limits.memory
    msg := sprintf("[resources.rego] container '%s' is missing resources.limits.memory", [c.name])
}

deny[msg] {
    input.kind == "Deployment"
    some i
    c := input.spec.template.spec.containers[i]
    not c.resources.requests.cpu
    msg := sprintf("[resources.rego] container '%s' is missing resources.requests.cpu", [c.name])
}

deny[msg] {
    input.kind == "Deployment"
    some i
    c := input.spec.template.spec.containers[i]
    not c.resources.requests.memory
    msg := sprintf("[resources.rego] container '%s' is missing resources.requests.memory", [c.name])
}

# Disallow unbounded limits
deny[msg] {
    input.kind == "Deployment"
    some i
    c := input.spec.template.spec.containers[i]
    c.resources.limits.memory == "0"
    msg := sprintf("[resources.rego] container '%s' has memory limit '0'", [c.name])
}
