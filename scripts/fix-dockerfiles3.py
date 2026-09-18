import os

go_services = ['search-service', 'observability-service', 'dynamic-model-service',
               'auth-service', 'email-service', 'meta-capi-service', 'lead-service',
               'crm-service', 'notification-service', 'tenant-service', 'analytics-service',
               'billing-service', 'landing-service', 'chat-engine', 'webrtc-sfu']

for svc in go_services:
    dockerfile = os.path.join(r'C:\code\RINCO\services', svc, 'Dockerfile')
    with open(dockerfile, 'r') as f:
        content = f.read()
    new = content.replace('COPY go.mod go.sum* ./', f'COPY services/{svc}/go.mod services/{svc}/go.sum* ./')
    with open(dockerfile, 'w') as f:
        f.write(new)
    print(f'Fixed {svc}')

print('Done')
