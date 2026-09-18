import re

with open(r'C:\code\RINCO\infra\docker-compose.services.yml', 'r') as f:
    content = f.read()

new = re.sub(
    r'build: C:/code/RINCO/frontend/(\w[\w-]+)',
    r'build:\n      context: C:/code/RINCO\n      dockerfile: frontend/\1/Dockerfile',
    content
)

with open(r'C:\code\RINCO\infra\docker-compose.services.yml', 'w') as f:
    f.write(new)
print('Done')
