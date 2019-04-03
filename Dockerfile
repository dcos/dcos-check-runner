FROM golang:1.12.1

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN go get github.com/stretchr/testify \
    && go get -u github.com/client9/misspell/cmd/misspell \
    && go get -u golang.org/x/tools/cmd/goimports \
    && go get -u golang.org/x/lint/golint

WORKDIR /dcos-check-runner
