CREATE TABLE mode_classes (
  mode TEXT NOT NULL REFERENCES imaging_modes (id),
  class TEXT NOT NULL REFERENCES classes (name),
  PRIMARY KEY (mode, class)
);
