FROM openjdk:17-jdk-slim

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    unzip \
    python3 \
    python3-pip \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install ROBOT
WORKDIR /opt
RUN wget https://github.com/ontodev/robot/releases/download/v1.9.5/robot.jar
RUN wget https://raw.githubusercontent.com/ontodev/robot/master/bin/robot
RUN chmod +x robot
RUN mv robot /usr/local/bin/
ENV ROBOT_JAVA_ARGS="-Xmx4g"

# Install additional tools
RUN pip3 install requests rdflib owlready2

# Create workspace
WORKDIR /workspace
COPY semantic/robot-scripts/ ./scripts/
COPY semantic/robot-configs/ ./configs/

# Set up entrypoint
COPY semantic/robot-entrypoint.sh ./entrypoint.sh
RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
CMD ["validate"]