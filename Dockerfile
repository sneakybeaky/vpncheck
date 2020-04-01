FROM alpine:latest as certs
RUN apk --update add ca-certificates

# build stage
FROM golang:1.14 AS build-env

# All these steps will be cached
RUN mkdir /vpnck
WORKDIR /vpnck
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download
# COPY the source code as the last step
COPY . .

# Run tests
RUN go test -v ./...

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/vpnck cmd/vpnck.go

# final stage
FROM alpine:latest

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=build-env /go/bin/vpnck /app/
COPY --from=build-env /vpnck/templates /app/templates
ENTRYPOINT ./vpnck
