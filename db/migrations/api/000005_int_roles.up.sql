CREATE TEMP TABLE tmp_roles_enum (
	id INT,
	name VARCHAR(32)
);
INSERT INTO tmp_roles_enum (id, name)
VALUES
	(1, 'admin'),
	(2, 'default'),
	(3, 'family'),
	(4, 'tanya');

CREATE OR REPLACE FUNCTION str_role_as_int(str_role VARCHAR(32))
RETURNS INT
LANGUAGE plpgsql AS $$
DECLARE
  int_role INT;
BEGIN
	SELECT id INTO int_role
	FROM tmp_roles_enum
	WHERE name = str_role;
	return int_role;
END;
$$;

CREATE OR REPLACE FUNCTION str_roles_as_int(str_roles_arr VARCHAR(32)[])
RETURNS INT[]
LANGUAGE plpgsql AS $$
DECLARE
  e VARCHAR(32);
  int_roles INT[];
BEGIN
	FOREACH e IN ARRAY str_roles_arr LOOP
		int_roles := int_roles || str_role_as_int(e);
	END LOOP;
	RETURN int_roles;
END;
$$;

UPDATE "user" SET roles = str_roles_as_int(roles)::VARCHAR(32)[];

ALTER TABLE "user"
ALTER COLUMN roles TYPE INT[]
USING (roles::INT[]);

DROP TABLE tmp_roles_enum;
DROP FUNCTION str_role_as_int;
DROP FUNCTION str_roles_as_int;
