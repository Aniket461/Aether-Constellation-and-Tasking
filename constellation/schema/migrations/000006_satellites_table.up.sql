CREATE TABLE satellites (
  name TEXT PRIMARY KEY,
  catalog INTEGER NOT NULL UNIQUE,
  plane TEXT NOT NULL REFERENCES planes (name),
  class TEXT NOT NULL REFERENCES classes (name),
  look TEXT NOT NULL CHECK (look IN ('right', 'left', 'either')),
  line1 TEXT NOT NULL,
  line2 TEXT NOT NULL,
  sort_order INTEGER NOT NULL UNIQUE
);
