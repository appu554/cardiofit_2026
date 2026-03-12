# Dockerfile for ROBOT Tool
# Purpose: Ontology merge, reasoning, validation
# Base: Java 11 (required for ROBOT v1.9.5)

FROM eclipse-temurin:11-jdk-jammy

LABEL maintainer="kb7-team@cardiofit.ai"
LABEL description="ROBOT Tool for ontology operations"
LABEL version="1.9.5"

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    unzip \
    jq \
    file \
    && rm -rf /var/lib/apt/lists/*

# Download ROBOT v1.9.5
WORKDIR /app
RUN curl -fsSL -o robot.jar \
    https://github.com/ontodev/robot/releases/download/v1.9.5/robot.jar \
    && echo "Verifying robot.jar..." \
    && file robot.jar \
    && test -s robot.jar \
    && jar tf robot.jar > /dev/null \
    && echo "✅ robot.jar valid"

# Create robot wrapper script (since v1.9.5 doesn't include one)
RUN echo '#!/bin/bash' > robot \
    && echo 'java ${ROBOT_JAVA_ARGS} -jar /app/robot.jar "$@"' >> robot \
    && chmod +x robot \
    && echo "✅ robot wrapper created"

# Verify download and create checksum
RUN sha256sum robot.jar > robot-checksum.txt \
    && cat robot-checksum.txt

# Copy scripts
COPY scripts/merge-ontologies.sh /app/scripts/
COPY scripts/run-reasoning.sh /app/scripts/
COPY scripts/package-kernel.sh /app/scripts/
COPY scripts/validate-uri-alignment.sh /app/scripts/
RUN chmod +x /app/scripts/*.sh

# Create workspace directories
RUN mkdir -p /workspace /queries

# Set ROBOT environment
ENV ROBOT_JAR=/app/robot.jar
ENV PATH="/app:${PATH}"

# Default JVM options (overridable)
ENV ROBOT_JAVA_ARGS="-Xmx8G -XX:+UseG1GC"

WORKDIR /workspace

ENTRYPOINT ["/bin/bash"]
CMD ["-c", "echo 'ROBOT Tool ready. Use /app/scripts/ for operations'"]
