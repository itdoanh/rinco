import os

services_root = r'C:\code\RINCO\services'
mapping = {
    'search-service': 'COPY ./services/search-service/ .',
    'observability-service': 'COPY ./services/observability-service/ .',
    'dynamic-model-service': 'COPY ./services/dynamic-model-service/ .',
    'auth-service': 'COPY ./services/auth-service/ .',
    'email-service': 'COPY ./services/email-service/ .',
    'meta-capi-service': 'COPY ./services/meta-capi-service/ .',
    'lead-service': 'COPY ./services/lead-service/ .',
    'crm-service': 'COPY ./services/crm-service/ .',
    'notification-service': 'COPY ./services/notification-service/ .',
    'tenant-service': 'COPY ./services/tenant-service/ .',
    'analytics-service': 'COPY ./services/analytics-service/ .',
    'billing-service': 'COPY ./services/billing-service/ .',
}

for svc, replacement in mapping.items():
    dockerfile = os.path.join(services_root, svc, 'Dockerfile')
    with open(dockerfile, 'r') as f:
        content = f.read()
    new = content.replace('COPY . .', replacement).replace('COPY ./ ./', replacement)
    with open(dockerfile, 'w') as f:
        f.write(new)
    print(f'Fixed {svc}')

# Fix frontends
frontends = ['landing', 'admin-portal', 'tenant-site', 'meeting-ui']
for fe in frontends:
    dockerfile = os.path.join(r'C:\code\RINCO\frontend', fe, 'Dockerfile')
    with open(dockerfile, 'r') as f:
        content = f.read()
    new = content.replace('COPY . .', f'COPY ./frontend/{fe}/ .')
    with open(dockerfile, 'w') as f:
        f.write(new)
    print(f'Fixed {fe}')

print('Done')
