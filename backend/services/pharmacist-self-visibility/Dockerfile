# pharmacist-self-visibility service image
#
# Runtime requirements (set via Kubernetes secrets / env):
#   VAIDSHALA_DSN — PostgreSQL connection string (required; server exits if absent)
#   JWT_SECRET    — HMAC-HS256 signing secret for bearer token verification (required)
#   PORT          — TCP port to listen on (optional; defaults to 8140)
#
# Build:   docker build -t pharmacist-self-visibility .
# Run:     docker run -e VAIDSHALA_DSN=<dsn> -e JWT_SECRET=<secret> -p 8140:8140 pharmacist-self-visibility

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /out/server ./cmd/server

FROM alpine:3.19
COPY --from=build /out/server /usr/local/bin/server
EXPOSE 8140
CMD ["/usr/local/bin/server"]
