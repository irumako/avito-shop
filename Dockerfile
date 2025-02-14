FROM golang:1.22

WORKDIR /avito-shop

COPY go.mod go.sum /avito-shop/
RUN go mod download && go mod verify

COPY . .
RUN go build -o /build ./cmd/api \
    && go clean -cache -modcache

EXPOSE 8080

CMD ["/build"]