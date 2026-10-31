CREATE TRIGGER mode_classes_version
  AFTER INSERT OR UPDATE OR DELETE ON mode_classes
  FOR EACH STATEMENT EXECUTE FUNCTION bump_catalog_version();
