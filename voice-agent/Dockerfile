FROM python:3.12-slim

WORKDIR /app

COPY pyproject.toml README.md ./
COPY src ./src

RUN python -m pip install --no-cache-dir -e .

ENV JARVIS_CORE_URL=http://backend:18080
ENV JARVIS_VOICE_AGENT_HOST=0.0.0.0
ENV JARVIS_VOICE_AGENT_PORT=18100

EXPOSE 18100

CMD ["python", "-m", "jarvis_voice_agent.server"]
