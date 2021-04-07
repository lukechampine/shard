FROM golang:alpine AS build
WORKDIR $GOPATH/src/github.com/lukechampine/shard
COPY . .
ENV CGO_ENABLED=0
RUN apk -U --no-cache add bash upx git gcc make \
    && make static \
    && upx /go/bin/shard

FROM scratch
COPY --from=build /go/bin/shard /shard
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/shard"]
