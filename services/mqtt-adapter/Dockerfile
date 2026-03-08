FROM golang:1.25 AS build
WORKDIR /app
COPY services/mqtt-adapter/go.mod services/mqtt-adapter/go.sum* ./
RUN go mod download
COPY services/mqtt-adapter/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/mqtt-adapter ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/mqtt-adapter /mqtt-adapter
EXPOSE 8083
USER nonroot:nonroot
ENTRYPOINT ["/mqtt-adapter"]