CREATE TEMP TABLE tmp_roles_enum (
	id int,
	name varchar(32)
);
INSERT INTO tmp_roles_enum (id, name)
VALUES
	(1, 'admin'),
	(2, 'default'),
	(3, 'family'),
	(4, 'tanya');


CREATE OR REPLACE FUNCTION int_role_as_str(int_role INT)
RETURNS VARCHAR(32)
LANGUAGE plpgsql AS $$
DECLARE
  str_role VARCHAR(32);
BEGIN
	SELECT name INTO str_role
	FROM tmp_roles_enum WHERE id = int_role;
	return str_role;
END;
$$;

CREATE OR REPLACE FUNCTION int_roles_as_str(int_roles_arr INT[])
RETURNS VARCHAR(32)[]
LANGUAGE plpgsql AS $$
DECLARE
  e INT;
  str_roles VARCHAR(32)[];
BEGIN
	FOREACH e IN ARRAY int_roles_arr LOOP
		str_roles := str_roles || int_role_as_str(e);
	END LOOP;
	RETURN str_roles;
END;
$$;

CREATE TEMP TABLE tmp_user AS SELECT * FROM "user";

UPDATE "user" SET roles = ARRAY[]::INT[];

ALTER TABLE "user"
ALTER COLUMN roles TYPE VARCHAR(32)[]
USING (roles::VARCHAR(32)[]);

UPDATE "user" u
   SET roles = int_roles_as_str(tu.roles)
  FROM tmp_user tu
 WHERE u.id = tu.id;

DROP TABLE tmp_roles_enum;
DROP TABLE tmp_user;
DROP FUNCTION int_role_as_str;
DROP FUNCTION int_roles_as_str;
