FROM golang:latest
MAINTAINER xtaci <daniel820313@gmail.com>
ENV GOBIN /go/bin
COPY . /go
WORKDIR /go
ENV GOPATH /go:/go/.godeps
RUN go install snowflake
RUN rm -rf pkg src .godeps
ENTRYPOINT /go/bin/snowflake
EXPOSE 50003
