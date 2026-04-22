FROM golang:1.26.2 AS builder

COPY . .

RUN cd cmd/tarunner && GOOS=linux GOARCH=amd64 go build . && cp tarunner /

FROM redhat/ubi10-micro

COPY --from=builder --chmod=755 /tarunner /tarunner

ENTRYPOINT ["/tarunner"]
