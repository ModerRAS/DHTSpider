FROM golang:1.16-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
RUN go env | grep GOPROXY

RUN go mod download

COPY *.go ./

RUN go build -o /DHTSpider

CMD [ "/DHTSpider" ]