%FROM gliderlabs/alpine:3.1
ENTRYPOINT ["/bin/tcp-forward-proxy"]

COPY . /go/src/github.com/mikroio/tcp-forward-proxy
RUN apk-install -t build-deps go git mercurial \
	&& cd /go/src/github.com/mikroio/tcp-forward-proxy \
	&& export GOPATH=/go \
	&& go get \
	&& go build -ldflags "-X main.Version $(cat VERSION)" -o /bin/tcp-forward-proxy \
	&& rm -rf /go \
	&& apk del --purge build-deps
