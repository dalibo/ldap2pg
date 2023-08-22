-- Execute this before executing readme/ldap2pg.yml to have a few changes.
DROP ROLE IF EXISTS "charles";
CREATE ROLE "omar";
ALTER ROLE "alain" WITH NOLOGIN CONNECTION LIMIT -1;
