# Functionnal tests

Functionnal tests tends to integrate ldap2pg in real world: execution in target
system, installation from rpm package, no mocks.

Run `make clean rpm tests` to recreate rpm and test env.

On error, the container wait forever. Either enter the container with `make
debug` or kill it with ^C. To reduce dev loop, just `pip install -e .` to use
wip code rather than rpm version.
