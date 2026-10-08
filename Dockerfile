FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api && \
    CGO_ENABLED=0 go build -o /bin/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -o /bin/mockprovider ./cmd/mockprovider && \
    CGO_ENABLED=0 go build -o /bin/reconcile ./cmd/reconcile

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/api /bin/worker /bin/mockprovider /bin/reconcile /usr/local/bin/
WORKDIR /app
