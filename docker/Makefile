# This target helps building local image when editing Dockerfil. http_proxy arg
# is transmitted to build env to enable APT cacher.
all:
	docker build --build-arg http_proxy -t dalibo/ldap2pg:local .

