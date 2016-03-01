FROM r.fds.so:5000/golang1.5.3
ENV GOBIN /go/bin
COPY . /go
WORKDIR /go
ENV GOPATH /go:/go/.godeps
RUN go install snowflake
RUN rm -rf pkg src .godeps
ENTRYPOINT /go/bin/snowflake
EXPOSE 50003
