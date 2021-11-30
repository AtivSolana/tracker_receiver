FROM golang:1.16-alpine
RUN apk update && apk add bash

WORKDIR /receiver

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go build -o /docker-gs-ping

RUN chmod +x ./wait-for-it.sh ./docker-entrypoint.sh

ENTRYPOINT ["./docker-entrypoint.sh"]

CMD [ "/docker-gs-ping" ]