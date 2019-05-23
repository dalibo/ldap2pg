FROM debian:buster-slim

RUN set -ex; \
    apt-get update -y; \
    apt-get install -y --no-install-recommends \
        python3=3.7.2-1 \
        python3-ldap=3.1.0-2 \
        python3-pip=18.1-5 \
        python3-psycopg2=2.7.7-1 \
        python3-setuptools=40.8.0-1 \
        python3-yaml=3.13-2 \
    ; \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*; \
    :

RUN set -ex; \
    pip3 --no-cache-dir install --no-deps ldap2pg==4.18; \
    ldap2pg --version; \
    :

ENTRYPOINT ["ldap2pg"]
WORKDIR /workspace