CREATE TRIGGER imaging_modes_version
  AFTER INSERT OR UPDATE OR DELETE ON imaging_modes
  FOR EACH STATEMENT EXECUTE FUNCTION bump_catalog_version();
