FROM golang:1.19

EXPOSE 8123

WORKDIR /usr/src/app

RUN mkdir /etc/inventa

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/ ./...

USER 1000:1000
CMD ["/usr/local/bin/inventa", "-c", "/etc/inventa/config.yaml"]