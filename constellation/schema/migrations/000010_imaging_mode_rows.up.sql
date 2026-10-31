INSERT INTO imaging_modes (
  id, label, seconds, range_km, azimuth_km, perform_min, perform_max, request_min, request_max, sort_order
) VALUES
  ('SPOTLIGHT', 'Spot', 10, 5, 5, 20, 40, 10, 45, 1),
  ('SPOTLIGHT_FINE', 'Spot Fine', 15, 5, 5, 20, 40, 10, 45, 2),
  ('SPOTLIGHT_EXTENDED_AREA', 'Spot Extended Area', 10, 15, 15, 20, 40, 10, 45, 3),
  ('SPOTLIGHT_EXTENDED_DWELL', 'Dwell', 25, 5, 5, 20, 40, 10, 45, 4),
  ('SPOTLIGHT_EXTENDED_DWELL_FINE', 'Dwell Fine', 25, 5, 5, 20, 40, 10, 45, 5),
  ('SPOTLIGHT_EXTENDED_DWELL_PRECISE', 'Dwell Precise', 25, 5, 5, 20, 33, 10, 45, 6),
  ('STRIPMAP', 'Strip', 10, 30, 50, 15, 35, 10, 45, 7),
  ('SCAN', 'Scan', 15, 100, 100, 21, 29, 21, 29, 8),
  ('SCAN_WIDE', 'Scan Wide', 42, 200, 300, 21, 26, 21, 26, 9);
