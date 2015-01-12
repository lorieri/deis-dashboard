FROM google/golang

RUN go get gopkg.in/redis.v2 github.com/coreos/go-etcd/etcd

ADD . /

RUN go build web.go

EXPOSE 6969

ENTRYPOINT ["/web"]
