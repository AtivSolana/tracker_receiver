FROM golang:1.16-alpine

WORKDIR /receiver

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /docker-gs-ping

CMD [ "/docker-gs-ping" ]