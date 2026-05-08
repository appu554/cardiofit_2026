# kb-cql-runtime container: Spring Boot + HAPI FHIR
# Plan 0.5 Task 1 - scaffold only; runtime CQL evaluation arrives in Tasks 3-5.

FROM eclipse-temurin:17-jdk-alpine AS builder
WORKDIR /build
COPY pom.xml .
COPY src ./src
RUN apk add --no-cache maven && mvn -B package -DskipTests

FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY --from=builder /build/target/kb-cql-runtime-*.jar app.jar
EXPOSE 8140
ENTRYPOINT ["java","-jar","app.jar"]
