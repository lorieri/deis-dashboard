FROM google/golang

RUN go get gopkg.in/redis.v2

ADD . /

RUN go build web.go

EXPOSE 6969

ENTRYPOINT ["/web"]
