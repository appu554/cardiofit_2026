FROM python:3.11-slim

LABEL maintainer="CardioFit Clinical Synthesis Hub"
LABEL description="GitHub Dispatcher for Australian Knowledge Factory"
LABEL region="au"

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY main.py .

CMD ["python", "main.py"]
