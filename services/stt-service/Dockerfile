FROM nvidia/cuda:12.3.0-runtime-ubuntu22.04
WORKDIR /app
RUN apt-get update && apt-get install -y python3.12 python3-pip && rm -rf /var/lib/apt/lists/*
COPY services/stt-service/requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt
COPY services/stt-service/main.py .
EXPOSE 8093
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8093", "--workers", "1"]
