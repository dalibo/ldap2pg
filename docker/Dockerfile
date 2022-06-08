FROM debian:bullseye-slim

ARG LDAP2PG_VERSION

RUN set -ex; \
    apt-get update -y; \
    apt-get install -y --no-install-recommends \
        libsasl2-modules \
        python3 \
        python3-ldap \
        python3-pip \
        python3-psycopg2 \
        python3-setuptools \
        python3-yaml \
    ; \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
    :

RUN set -ex; \
    pip3 --no-cache-dir install --no-deps ldap2pg${LDAP2PG_VERSION:+==${LDAP2PG_VERSION}}; \
    ldap2pg --version; \
    :

# Set LANG for execution order of entrypoint.d run parts.
ENV LANG en_US.utf8
WORKDIR /workspace

COPY docker-entrypoint.sh /usr/local/bin
RUN mkdir /docker-entrypoint.d
ENTRYPOINT ["docker-entrypoint.sh"]
