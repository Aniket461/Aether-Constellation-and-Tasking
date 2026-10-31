CREATE TABLE imaging_modes (
  id TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  seconds INTEGER NOT NULL CHECK (seconds > 0),
  range_km DOUBLE PRECISION NOT NULL CHECK (range_km > 0),
  azimuth_km DOUBLE PRECISION NOT NULL CHECK (azimuth_km > 0),
  perform_min DOUBLE PRECISION NOT NULL,
  perform_max DOUBLE PRECISION NOT NULL,
  request_min DOUBLE PRECISION NOT NULL,
  request_max DOUBLE PRECISION NOT NULL,
  sort_order INTEGER NOT NULL UNIQUE,
  CONSTRAINT imaging_modes_perform CHECK (perform_min <= perform_max),
  CONSTRAINT imaging_modes_request CHECK (request_min <= request_max)
);
