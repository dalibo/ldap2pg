# This target helps building local image when editing Dockerfile. http_proxy arg
# is transmitted to build env to enable APT cacher.
build:
	docker build --build-arg http_proxy -t dalibo/ldap2pg:local .

pentest:
	docker run --rm dalibo/ldap2pg:local --version
