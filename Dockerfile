FROM golang:1.9
ADD . /go/src/github.com/palicao/docker-executor
RUN go install github.com/palicao/docker-executor
ADD ./config.yaml /etc/docker-executor/config.yaml
RUN chmod +x /go/bin/docker-executor
ENTRYPOINT /go/bin/docker-executor -config /etc/docker-executor/config.yaml
EXPOSE 8080