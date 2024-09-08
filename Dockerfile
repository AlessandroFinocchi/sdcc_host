FROM golang:1.21-alpine

RUN apk update

WORKDIR /app

COPY go.mod ./

RUN go mod download

COPY . .

RUN apk add iproute2 iputils

EXPOSE 50152 50153 50154

RUN go build -o /sdcc_host

RUN mkdir -p /data

ENTRYPOINT ["sh", "host_setup.sh"]