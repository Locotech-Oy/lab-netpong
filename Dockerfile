FROM golang:1.18 as builder
LABEL stage=builder

WORKDIR $GOPATH/go/
COPY go/go.mod .
COPY go/go.sum .
RUN go mod download

COPY go/ .
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/netpong

FROM gcr.io/distroless/base-debian11 as final
	# checkov:skip=CKV_DOCKER_7: base image does not provide tagged alternative

LABEL maintainer="jens.wegar@locotech.fi"
USER nonroot:nonroot

COPY --from=builder --chown=nonroot:nonroot /go/bin/netpong /netpong

CMD [ "/netpong" ]