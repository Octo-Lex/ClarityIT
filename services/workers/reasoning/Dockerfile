FROM python:3.12-slim

RUN groupadd -r clarityit && useradd -r -g clarityit clarityit
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN chown -R clarityit:clarityit /app
USER clarityit

# Worker has NO database, NATS, Redis, or MinIO credentials
# It communicates ONLY through the Go API
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://clarityit-api:8765/health')" || exit 1
CMD ["python", "-m", "worker"]
