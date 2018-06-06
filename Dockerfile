FROM golang:alpine as build

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

FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
RUN cd /app
COPY --from=build /usr/local/go/src/github.com/mit-dci/lit/lit /app/bin/lit
COPY --from=build /usr/local/go/src/github.com/mit-dci/lit/cmd/lit-af/lit-af /app/bin/lit-af

EXPOSE 8001

CMD ["bin/lit"]