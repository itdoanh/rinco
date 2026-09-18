import os

# Mapping of service -> requirements.txt path (relative to monorepo root)
service_req_mapping = {
    'stt-service': 'services/stt-service/requirements.txt',
    'recording-service': 'services/recording-service/requirements.txt',
    'ai-sre': 'services/ai-sre/requirements.txt',
    'rag-chatbot': 'services/rag-chatbot/requirements.txt',
    'lead-scoring': 'services/lead-scoring/requirements.txt',
}

for svc, req_path in service_req_mapping.items():
    dockerfile = os.path.join(r'C:\code\RINCO\services', svc, 'Dockerfile')
    with open(dockerfile, 'r') as f:
        content = f.read()
    new = content.replace('COPY requirements.txt .', f'COPY {req_path} .')
    with open(dockerfile, 'w') as f:
        f.write(new)
    print(f'Fixed {svc}')

# Also fix the go services that use COPY go.mod/go.sum at root
go_services = ['search-service', 'observability-service', 'dynamic-model-service',
               'auth-service', 'email-service', 'meta-capi-service', 'lead-service',
               'crm-service', 'notification-service', 'tenant-service', 'analytics-service',
               'billing-service']

# These have COPY go.mod ./ which is fine if go.mod is at root, but actually the go.mod
# is INSIDE each service. Let me check
for svc in go_services:
    svc_path = os.path.join(r'C:\code\RINCO\services', svc)
    has_gomod = os.path.exists(os.path.join(svc_path, 'go.mod'))
    print(f'{svc}: go.mod exists={has_gomod}')

print('Done')
