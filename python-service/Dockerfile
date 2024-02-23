# https://hub.docker.com/_/python
FROM python:3.12-slim as builder


WORKDIR /app
COPY requirements.txt ./

RUN pip install uv && uv venv
RUN . .venv/bin/activate && uv pip sync requirements.txt

FROM python:3.12-slim

ENV PYTHONUNBUFFERED True
ENV APP_HOME /app
WORKDIR $APP_HOME

COPY --from=builder /app/.venv /app/.venv
COPY pysvc.py ./

ENV PORT 8080
ENV PATH=/app/.venv/bin
CMD uvicorn pysvc:app --host 0 --port $PORT
