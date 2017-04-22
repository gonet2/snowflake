FROM golang:latest
MAINTAINER xtaci <daniel820313@gmail.com>
COPY . /go/src/snowflake
RUN go install snowflake
ENTRYPOINT ["/go/bin/snowflake"]
EXPOSE 10000 6060
