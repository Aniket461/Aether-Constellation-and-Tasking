CREATE FUNCTION bump_catalog_version() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  UPDATE constellation SET version = version + 1 WHERE id;
  RETURN NULL;
END;
$$;
