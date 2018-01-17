srcdir = .
PG_CONFIG ?= pg_config
PGXS := $(shell $(PG_CONFIG) --pgxs)
CFLAGS = -I$(shell $(PG_CONFIG) --includedir-server) $(CPPFLAGS)
include $(PGXS)
override CPPFLAGS := $(CPPFLAGS) -I$(shell $(PG_CONFIG) --includedir)
override LDFLAGS :=  $(shell $(PG_CONFIG) --ldflags) $(LDFLAGS)
override LDFLAGS_EX :=  $(shell $(PG_CONFIG) --ldflags_ex) $(LDFLAGS_EX)
override CPPFLAGS := -I$(libpq_srcdir) $(CPPFLAGS)
override LDLIBS := $(libpq_pgport) $(LDLIBS)


all: pg_dumpacl

pg_dumpacl: pg_dumpacl.o


install: all installdirs
	$(INSTALL_PROGRAM) pg_dumpacl$(X) '$(DESTDIR)$(bindir)'/pg_dumpacl$(X)

installdirs:
	$(MKDIR_P) '$(DESTDIR)$(bindir)'

uninstall:
	rm -f $(addprefix '$(DESTDIR)$(bindir)'/, pg_dump_acl$(X)

clean distclean maintainer-clean:
	rm -f pg_dumpacl pg_dumpacl.o

PGDG=https://download.postgresql.org/pub/repos/yum
rpms:
	PGDG_RPM=$(PGDG)/10/redhat/rhel-7-x86_64/pgdg-centos10-10-2.noarch.rpm PGVERSION=10 \
		docker-compose run --rm rpm
	PGDG_RPM=$(PGDG)/9.6/redhat/rhel-7-x86_64/pgdg-centos96-9.6-3.noarch.rpm PGVERSION=9.6 \
		docker-compose run --rm rpm
	PGDG_RPM=$(PGDG)/9.5/redhat/rhel-7-x86_64/pgdg-centos95-9.5-3.noarch.rpm PGVERSION=9.5 \
		docker-compose run --rm rpm
	PGDG_RPM=$(PGDG)/9.4/redhat/rhel-7-x86_64/pgdg-centos94-9.4-3.noarch.rpm PGVERSION=9.4 \
		docker-compose run --rm rpm
	PGDG_RPM=$(PGDG)/9.3/redhat/rhel-7-x86_64/pgdg-centos93-9.3-3.noarch.rpm PGVERSION=9.3 \
		docker-compose run --rm rpm
