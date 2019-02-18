FROM debian:buster-slim

RUN set -ex; \
    apt-get update -y; \
    apt-get install -y --no-install-recommends \
        python3 \
        python3-ldap \
        python3-pip \
        python3-psycopg2 \
        python3-setuptools \
        python3-yaml \
    ; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
    :

RUN set -ex; \
    pip3 --no-cache-dir install --no-deps ldap2pg; \
    ldap2pg --version; \
    :

ENTRYPOINT ["ldap2pg"]
WORKDIR /workspace
