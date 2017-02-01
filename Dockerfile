FROM golang:1.8-alpine

COPY conf.json /go/bin/

RUN apk add --no-cache git && go get git.esheavyindustries.com/esell/hoptocopter

CMD cd /go/bin && ./hoptocopter

EXPOSE 8080
