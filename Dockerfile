FROM golang:alpine

WORKDIR /app

COPY . /app

ENV GOPATH /app

RUN go build -o user-stats

EXPOSE 8090

CMD ["/app/user-stats", "-config", "./config.yml"]