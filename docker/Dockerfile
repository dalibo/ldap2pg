FROM goreleaser/goreleaser:v2.6.1 AS builder

WORKDIR /workspace
COPY . .
# Clean directory of untracked files, even those in developer ~/.config/git/ignore
RUN git clean -df
# Simulate release --auto-snapshot for build command.
RUN if git describe --tags --exact-match; then \
    goreleaser build --clean --single-target; \
    else \
    goreleaser build --clean --single-target --snapshot ; \
    fi

FROM alpine:3.18

# Set LANG for execution order of entrypoint.d run parts.
ENV LANG en_US.utf8
WORKDIR /workspace

COPY --from=builder /workspace/dist/ldap2pg_linux_amd64_v1/ldap2pg /usr/bin/ldap2pg

RUN apk add --no-cache bash openldap-clients
RUN mkdir /docker-entrypoint.d
COPY docker/docker-entrypoint.sh /usr/local/bin
ENTRYPOINT ["docker-entrypoint.sh"]

# Smoketest
RUN /usr/local/bin/docker-entrypoint.sh --version
