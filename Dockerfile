FROM golang:alpine as builder

WORKDIR $GOPATH/go/
COPY go/* .
RUN go build -ldflags "-w" -o /go/app

FROM golang:alpine

COPY --from=builder /go/app /go/app

CMD ["/go/app"]