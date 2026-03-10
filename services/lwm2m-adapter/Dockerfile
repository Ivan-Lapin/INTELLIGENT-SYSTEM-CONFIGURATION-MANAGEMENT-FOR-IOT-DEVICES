FROM golang:1.25 AS build
WORKDIR /app
COPY services/lwm2m-adapter/go.mod services/lwm2m-adapter/go.sum* ./
RUN go mod download
COPY services/lwm2m-adapter/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/lwm2m-adapter ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bin/lwm2m-adapter /lwm2m-adapter
EXPOSE 8088
USER nonroot:nonroot
ENTRYPOINT ["/lwm2m-adapter"]