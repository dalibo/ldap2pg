# Contributing

## Releasing a new version

- Increment version in `pg_dumpacl.spec`
- Add a changelog entry in `pg_dumpacl.spec`
- Commit changes in `master` with message `Version X.Y`.
- Build rpms with `make rpms`. RPM files are in `rpm/x86_64`.
