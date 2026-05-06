# Wave 2.7 SKELETON Dockerfile.
# Builds the substrate-pipeline jar with Maven and runs it on JRE 17.
# V1 work: switch to a multi-stage build with a slimmer JRE base, add
# Prometheus JMX exporter agent, and pin the JVM flags.

FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /workspace
COPY pom.xml .
COPY src ./src
RUN mvn -B -DskipTests package

FROM eclipse-temurin:17-jre
WORKDIR /app
COPY --from=build /workspace/target/substrate-pipeline.jar /app/substrate-pipeline.jar

# TODO(wave-2.7-runtime): expose Prometheus metrics port, add HEALTHCHECK,
# pin JVM heap/GC flags from load-test results.
ENTRYPOINT ["java", "-jar", "/app/substrate-pipeline.jar"]
