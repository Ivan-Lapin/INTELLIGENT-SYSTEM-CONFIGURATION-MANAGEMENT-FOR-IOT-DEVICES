FROM golang:1.25 AS build
WORKDIR /app
COPY services/config-service/go.mod services/config-service/go.sum* ./
RUN go mod download
COPY services/config-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/config-service ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/config-service /config-service
EXPOSE 8082
USER nonroot:nonroot
ENTRYPOINT ["/config-service"]