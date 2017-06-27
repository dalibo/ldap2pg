DROP ROLE IF EXISTS spurious;
DROP ROLE IF EXISTS alice;
DROP ROLE IF EXISTS bob;
DROP ROLE IF EXISTS foo;
DROP ROLE IF EXISTS bar;
-- Create a spurious user, for DROP.
CREATE ROLE spurious WITH LOGIN;
-- Create alice superuser without login, for ALTER.
CREATE ROLE alice WITH SUPERUSER NOLOGIN;
