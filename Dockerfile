FROM golang:latest
MAINTAINER xtaci <daniel820313@gmail.com>
ENV GOBIN /go/bin
COPY . /go
WORKDIR /go
RUN go install snowflake
RUN rm -rf pkg src
ENTRYPOINT /go/bin/snowflake
EXPOSE 50003
