FROM golang:1.25 AS build
WORKDIR /app
COPY services/telemetry-ingestor/go.mod services/telemetry-ingestor/go.sum* ./
RUN go mod download
COPY services/telemetry-ingestor/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/telemetry-ingestor ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/telemetry-ingestor /telemetry-ingestor
EXPOSE 8085
USER nonroot:nonroot
ENTRYPOINT ["/telemetry-ingestor"]