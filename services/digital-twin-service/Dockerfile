FROM golang:1.25 AS build
WORKDIR /app
COPY services/digital-twin-service/go.mod services/digital-twin-service/go.sum* ./
RUN go mod download
COPY services/digital-twin-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/digital-twin-service ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/digital-twin-service /digital-twin-service
EXPOSE 8087
USER nonroot:nonroot
ENTRYPOINT ["/digital-twin-service"]