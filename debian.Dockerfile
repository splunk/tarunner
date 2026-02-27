# This Dockerfile is used by the integration example and installs additional software to try out the Splunk_TA_nix addon.
from golang:1.25.7 as builder

COPY . .

RUN cd cmd/tarunner && GOOS=linux GOARCH=amd64 go build . && cp tarunner /

from debian:13

COPY --from=builder --chmod=755 /tarunner /tarunner

RUN apt-get update && apt-get install -y net-tools lastlog2 ntpsec-ntpdate lsof sysstat auditd

RUN mkdir -p /var/run/splunk

ENTRYPOINT ["/tarunner"]
