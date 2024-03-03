FROM golang:1.19-alpine as builder
ARG UPX_VERSION=3.95

WORKDIR /app

# Cache the fetched Go packages
RUN apk add --no-cache gcc git musl-dev
COPY ./go.mod ./go.sum ./
RUN go mod download

# Then build the binary
COPY ./ ./

RUN go build

FROM alpine:3.9 as release
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/registrywatcher ./
# in practice this Dockerfile should never be run without interpolating a config file inside ./config

CMD ["/app/registrywatcher"]
