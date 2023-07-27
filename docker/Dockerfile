FROM goreleaser/goreleaser:v1.19.2 AS builder

WORKDIR /workspace
COPY . .
RUN goreleaser build --clean --snapshot --single-target

FROM alpine:3.18

# Set LANG for execution order of entrypoint.d run parts.
ENV LANG en_US.utf8
WORKDIR /workspace

COPY --from=builder /workspace/build/ldap2pg_linux_amd64_v1/ldap2pg /usr/bin/ldap2pg
RUN /usr/bin/ldap2pg --version

COPY docker/docker-entrypoint.sh /usr/local/bin
RUN mkdir /docker-entrypoint.d
ENTRYPOINT ["docker-entrypoint.sh"]
