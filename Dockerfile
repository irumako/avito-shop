FROM golang:1.22

RUN apt-get update && apt-get install -y \
    bash \
    curl \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz \
    | tar xvz && mv migrate /usr/local/bin/

WORKDIR /avito-shop

COPY go.mod go.sum /avito-shop/
RUN go mod download && go mod verify

COPY . .
RUN go build -o /build ./cmd/api \
    && go clean -cache -modcache

EXPOSE 8080

CMD ["/build"]