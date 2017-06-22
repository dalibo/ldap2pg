# Packaging `ldap2pg`

We provide recipe to build RPM package for `ldap2pg`. You require only Docker
and Docker Compose.

``` console
$ make rpm
...
rpm_1  | + chown --changes --recursive 1000:1000 dist/ build/
rpm_1  | changed ownership of 'dist/ldap2pg-0.1-1.src.rpm' from root:root to 1000:1000
rpm_1  | changed ownership of 'dist/ldap2pg-0.1-1.noarch.rpm' from root:root to 1000:1000
...
$
```

You will find `.rpm` package in `dist/`.
