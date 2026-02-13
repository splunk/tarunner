from golang:1.25.7 as builder

COPY . .

RUN cd cmd/tarunner && GOOS=linux GOARCH=amd64 go build . && cp tarunner /

from redhat/ubi10-micro

COPY --from=builder --chmod=755 /tarunner /tarunner

ENTRYPOINT ["/tarunner"]
