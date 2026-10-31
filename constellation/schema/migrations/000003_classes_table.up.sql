CREATE TABLE classes (
  name TEXT PRIMARY KEY,
  incidence_min DOUBLE PRECISION NOT NULL,
  incidence_max DOUBLE PRECISION NOT NULL,
  sort_order INTEGER NOT NULL UNIQUE,
  CONSTRAINT classes_incidence CHECK (incidence_min <= incidence_max)
);
