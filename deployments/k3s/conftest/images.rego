package main

# ============================================================
# OPA / conftest — only allow images from approved registries
# ============================================================

import rego.v1

approved_registries := {
    "ghcr.io/itdoanh",        # internal
    "ghcr.io/rinco",
    "docker.io/rinco",
    "docker.io/library",      # for system images we keep
    "public.ecr.aws",         # AWS public registry
    "gcr.io",                 # Google container registry
    "quay.io",                # Red Hat
    "mcr.microsoft.com",      # Microsoft
    "registry.k8s.io",        # K8s control-plane
}

deny[msg] {
    input.kind == "Deployment"
    some i
    image := input.spec.template.spec.containers[i].image
    not has_approved_registry(image)
    msg := sprintf("[images.rego] Image '%s' is NOT from an approved registry", [image])
}

deny[msg] {
    input.kind == "StatefulSet"
    some i
    image := input.spec.template.spec.containers[i].image
    not has_approved_registry(image)
    msg := sprintf("[images.rego] Image '%s' is NOT from an approved registry", [image])
}

deny[msg] {
    input.kind == "DaemonSet"
    some i
    image := input.spec.template.spec.containers[i].image
    not has_approved_registry(image)
    msg := sprintf("[images.rego] Image '%s' is NOT from an approved registry", [image])
}

# Helper: image must start with one of the approved registries
has_approved_registry(image) if {
    some r
    startswith(image, concat("/", [r, ""]))
    r in approved_registries
}

# Disallow `:latest` in production
deny[msg] {
    input.kind == "Deployment"
    some i
    image := input.spec.template.spec.containers[i].image
    endswith(image, ":latest")
    msg := sprintf("[images.rego] Image '%s' uses :latest tag — pin a version", [image])
}
