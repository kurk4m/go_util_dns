FROM golang:1.22-alpine

WORKDIR /app

COPY . .

RUN go build -o dnsserver .

EXPOSE 53/udp

CMD ["./dnsserver"]
