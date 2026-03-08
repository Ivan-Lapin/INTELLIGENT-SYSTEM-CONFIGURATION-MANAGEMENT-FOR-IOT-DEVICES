FROM golang:1.25 AS build
WORKDIR /app
COPY services/deployment-orchestrator/go.mod services/deployment-orchestrator/go.sum* ./
RUN go mod download
COPY services/deployment-orchestrator/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/deployment-orchestrator ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/deployment-orchestrator /deployment-orchestrator
EXPOSE 8084
USER nonroot:nonroot
ENTRYPOINT ["/deployment-orchestrator"]