# Official Python client (fluent-logger) for the functional tests.
FROM python:3.13-slim

RUN pip install --no-cache-dir fluent-logger==0.11.1

COPY python.client.py /client.py

ENTRYPOINT ["python", "/client.py"]
