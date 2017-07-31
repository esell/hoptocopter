FROM golang:1.8-alpine

WORKDIR /go/src/app
COPY . .

RUN go-wrapper download
RUN go-wrapper install

CMD ["go-wrapper", "run"]

EXPOSE 8080
