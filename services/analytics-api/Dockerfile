FROM python:3.12-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY pyproject.toml .
COPY src ./src

RUN pip install --no-cache-dir .

ENV PORT=3003
ENV SERVICE_NAME=analytics-api
EXPOSE 3003

USER nobody

CMD ["python", "-m", "analytics_api.main"]
