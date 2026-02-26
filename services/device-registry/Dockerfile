FROM golang:1.25 AS build
WORKDIR /app
COPY services/device-registry/go.mod services/device-registry/go.sum* ./
RUN go mod download
COPY services/device-registry/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/device-registry ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/device-registry /device-registry
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/device-registry"]