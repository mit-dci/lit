FROM golang:alpine

RUN apk update && apk upgrade && \
    apk add --no-cache bash git openssh gcc musl-dev
ENV GOROOT=/usr/local/go
RUN go get github.com/mit-dci/lit
RUN go get github.com/mit-dci/lit/cmd/lit-af
RUN rm -rf /usr/local/go/src/github.com/mit-dci/lit
COPY . /usr/local/go/src/github.com/mit-dci/lit
WORKDIR /usr/local/go/src/github.com/mit-dci/lit
RUN go build
WORKDIR /usr/local/go/src/github.com/mit-dci/lit/cmd/lit-af
RUN go build
EXPOSE 8001
ENTRYPOINT ["/usr/local/go/src/github.com/mit-dci/lit/lit"]