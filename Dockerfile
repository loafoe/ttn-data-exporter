FROM golang:1.20.4 as builder
WORKDIR /build
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build
COPY . .
RUN git rev-parse --short HEAD
RUN GIT_COMMIT=$(git rev-parse --short HEAD) && \
    CGO_ENABLED=0 go build -o ttn-data-exporter -ldflags "-X main.GitCommit=${GIT_COMMIT}"

FROM alpine:latest 
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
WORKDIR /app
COPY --from=builder /build/ttn-data-exporter /app
EXPOSE 8080
CMD ["/app/ttn-data-exporter"]
