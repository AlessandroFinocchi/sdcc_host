FROM golang:1.21-alpine

RUN apk update

WORKDIR /app/src/host

COPY go.mod ./

RUN go mod download

COPY . .

RUN apk add iproute2 iputils

EXPOSE 50152 50153 50154

RUN go build -o /sdcc_host

ENTRYPOINT ["sh", "host_setup.sh"]