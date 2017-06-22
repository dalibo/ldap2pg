# Functionnal tests

Functionnal tests tends to integrate ldap2pg in real world: execution in target
system, installation from rpm package, no mocks.

1. Create RPM package with `make -C ../../packaging/ distclean rpm trash`.
2. Run tests with `make tests`.

On error, the container wait forever. Either enter the container with `make
debug` or kill it with ^C.
