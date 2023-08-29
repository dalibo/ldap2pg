FROM goreleaser/goreleaser:v1.19.2 AS builder

WORKDIR /workspace
COPY . .
RUN goreleaser build --clean --snapshot --single-target

FROM alpine:3.18

# Set LANG for execution order of entrypoint.d run parts.
ENV LANG en_US.utf8
WORKDIR /workspace

COPY --from=builder /workspace/dist/ldap2pg_linux_amd64_v1/ldap2pg /usr/bin/ldap2pg

RUN apk add --no-cache bash
RUN mkdir /docker-entrypoint.d
COPY docker/docker-entrypoint.sh /usr/local/bin
ENTRYPOINT ["docker-entrypoint.sh"]

# Smoketest
RUN /usr/local/bin/docker-entrypoint.sh --version
