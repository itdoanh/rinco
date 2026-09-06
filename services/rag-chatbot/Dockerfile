FROM python:3.12-slim
WORKDIR /app
COPY services/rag-chatbot/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY services/rag-chatbot/main.py .
EXPOSE 8091
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8091", "--workers", "1"]
