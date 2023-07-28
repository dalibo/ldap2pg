-- Execute this one before executing ldap2pg.minimal.yml to have a few changes.
DROP DATABASE "appdb";
DROP DATABASE "olddb";
DROP ROLE "domitille";
CREATE ROLE "oscar" IN ROLE "ldap_roles";
ALTER ROLE "albert" WITH NOLOGIN;
