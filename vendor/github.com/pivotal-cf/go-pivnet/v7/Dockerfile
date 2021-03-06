FROM golang

RUN apt autoremove python -y

RUN apt-get update
RUN apt-get install jq -y

RUN go get -u github.com/onsi/ginkgo/ginkgo
RUN go get -u github.com/onsi/gomega/...
