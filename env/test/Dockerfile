FROM golang:1.20-alpine

COPY ./ /app
WORKDIR /app

RUN go mod download
RUN apk add --no-cache make
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.56.2
RUN go install -v github.com/incu6us/goimports-reviser/v3@latest

ENTRYPOINT ["tail", "-F", "/dev/null"]
