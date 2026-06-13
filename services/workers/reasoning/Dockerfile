FROM python:3.12-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .

# Worker has NO database, NATS, Redis, or MinIO credentials
# It communicates ONLY through the Go API
CMD ["python", "-m", "worker"]
