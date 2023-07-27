FROM golang:1.20-bullseye AS builder

WORKDIR /workspace
COPY . .
RUN go build -o /tmp/ldap2pg -trimpath -buildvcs -ldflags -s ./cmd/ldap2pg

FROM debian:bullseye-slim

# Set LANG for execution order of entrypoint.d run parts.
ENV LANG en_US.utf8
WORKDIR /workspace

COPY --from=builder /tmp/ldap2pg /usr/bin/ldap2pg
RUN /usr/bin/ldap2pg --version

COPY docker/docker-entrypoint.sh /usr/local/bin
RUN mkdir /docker-entrypoint.d
ENTRYPOINT ["docker-entrypoint.sh"]
