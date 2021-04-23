FROM golang:1.16-buster AS builder
WORKDIR /build/
COPY go.mod go.sum /build/
RUN go mod download
COPY . /build/
RUN go build -o workspace-ldap

FROM debian:buster
RUN apt-get update \
  && apt-get install -y ca-certificates \
  && rm -rf /var/lib/apt/lists/*
COPY --from=builder /build/workspace-ldap /
RUN setcap 'cap_net_bind_service=+ep' /workspace-ldap
ENTRYPOINT ["/workspace-ldap"]
