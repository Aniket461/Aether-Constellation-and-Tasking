# Databases

Aether Constellation and Tasking uses two Postgres databases on one server. Each service migrates its own database when it starts.

Constellation answers what the fleet can do, including the earliest free collection, and it owns each task. A task is the satellite's work: scheduled, imaging, downlink, processing, done, cancelled, or failed. Tasking answers the order. It stores `task_id` and the last status the constellation reported. It does not point at planes, classes, modes, or satellites, so a later edit to a mode leaves that order as it was.

Tasking does not keep a copy of the catalog and does not decide which satellite is free. A feasibility check calls `POST /api/acquisitions`. Placing an order calls `POST /api/tasks`. Both skip cancelled and failed tasks and treat scheduled, imaging, downlink, processing, and done tasks as busy. The busy interval is the imaging interval. Downlink and processing do not extend it. Constellation keeps the catalog in memory and reloads it when the catalog version changes. A new task does not change that version. Once a second, constellation writes the current status onto tasks that are still scheduled, imaging, downlink, or processing. Tasking polls those task ids and copies the status onto its orders.

## Constellation

The catalog version starts at 1. A change to planes, classes, imaging modes, mode classes, or satellites increments it.

### constellation

One row for the whole fleet.

| id | epoch | version |
| --- | --- | --- |
| true | 2026-10-29 00:00:00+00 | 1 |

### planes

| name | color | period_min | altitude_km | inclination_deg | raan_deg | mean_motion | ltan | sort_order |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Dawn | #7eb6ff | 95.03 | 520 | 97.4778 | 123.2760 | 15.15322385 | 06:00 | 1 |
| Morning | #e2b340 | 95.44 | 540 | 97.5544 | 190.7760 | 15.08756051 | 10:30 | 2 |
| Afternoon | #e07a5f | 95.86 | 560 | 97.6316 | 250.7760 | 15.02237003 | 14:30 | 3 |
| Dusk | #7dcea0 | 94.72 | 505 | 97.4207 | 303.2760 | 15.20278457 | 18:00 | 4 |

### classes

`incidence_min` and `incidence_max` are how far that bus can steer, in degrees.

| name | incidence_min | incidence_max | sort_order |
| --- | --- | --- | --- |
| Standard | 15 | 40 | 1 |
| Fine | 15 | 45 | 2 |
| Dwell | 15 | 40 | 3 |
| Wide | 15 | 35 | 4 |

### imaging_modes

A mode is one way to take the picture. `seconds` is how long the satellite stays busy. The perform columns are the performant incidence band. The request columns are the widest band a user can ask for. `range_km` and `azimuth_km` are the scene on the ground.

| id | label | seconds | range_km | azimuth_km | perform_min | perform_max | request_min | request_max |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| SPOTLIGHT | Spot | 10 | 5 | 5 | 20 | 40 | 10 | 45 |
| SPOTLIGHT_FINE | Spot Fine | 15 | 5 | 5 | 20 | 40 | 10 | 45 |
| SPOTLIGHT_EXTENDED_AREA | Spot Extended Area | 10 | 15 | 15 | 20 | 40 | 10 | 45 |
| SPOTLIGHT_EXTENDED_DWELL | Dwell | 25 | 5 | 5 | 20 | 40 | 10 | 45 |
| SPOTLIGHT_EXTENDED_DWELL_FINE | Dwell Fine | 25 | 5 | 5 | 20 | 40 | 10 | 45 |
| SPOTLIGHT_EXTENDED_DWELL_PRECISE | Dwell Precise | 25 | 5 | 5 | 20 | 33 | 10 | 45 |
| STRIPMAP | Strip | 10 | 30 | 50 | 15 | 35 | 10 | 45 |
| SCAN | Scan | 15 | 100 | 100 | 21 | 29 | 21 | 29 |
| SCAN_WIDE | Scan Wide | 42 | 200 | 300 | 21 | 26 | 21 | 26 |

Changing Spot from 10 seconds to 12 is one update on this table. The version becomes 2. No satellite row changes.

### mode_classes

Which classes can fly a mode. A satellite inherits the modes of its class.

| mode | class |
| --- | --- |
| SPOTLIGHT | Standard, Fine, Dwell |
| SPOTLIGHT_FINE | Fine |
| SPOTLIGHT_EXTENDED_AREA | Fine |
| SPOTLIGHT_EXTENDED_DWELL | Dwell |
| SPOTLIGHT_EXTENDED_DWELL_FINE | Dwell |
| SPOTLIGHT_EXTENDED_DWELL_PRECISE | Dwell |
| STRIPMAP | Standard, Fine, Dwell, Wide |
| SCAN | Standard, Fine, Wide |
| SCAN_WIDE | Wide |

Each class is its own row. Spotlight is three rows: Standard, Fine, and Dwell. Taking Scan away from Fine is deleting the Fine row for SCAN. The other Fine satellites follow, because they are not listed here themselves.

### satellites

Twelve satellites are seeded, three in each plane. Two of them:

| name | catalog | plane | class | look | sort_order |
| --- | --- | --- | --- | --- | --- |
| Aether-A1 | 95001 | Dawn | Standard | right | 1 |
| Aether-C2 | 95008 | Afternoon | Dwell | either | 8 |

Each row also stores `line1` and `line2`, the two lines of that satellite's TLE. Look is `right`, `left`, or `either`. Aether-A1 can fly Spotlight and Strip, because it is Standard. Aether-C2 can fly Spotlight, the dwell modes, and Strip, because it is Dwell. Neither can fly Scan Wide. That mode belongs to Wide.

### tasks

One row is one task. The row stays after the work finishes, so another portal can read it. The search skips scheduled, imaging, downlink, processing, and done intervals. Cancelled and failed rows stay and do not block the satellite. The migration creates the empty table. A row appears when someone calls `POST /api/tasks`, not at startup.

`status` is scheduled before imaging starts and imaging from the start until the end. After the imaging end the task is downlink, then processing, then done. `TASK_DOWNLINK_DELAY` and `TASK_PROCESSING_DELAY` are read when the constellation process starts. The defaults are 15s and 20s. `0s` skips that phase. A task already marked done stays done. A task whose whole pipeline is already over is stored as done immediately. Cancel sets cancelled and `cancelled_at`, and it is refused once imaging has ended. failed is allowed and nothing sets it yet. A one-second pass updates rows that are still scheduled, imaging, downlink, or processing. Reading a task uses the same rule, so the answer is current between ticks.

The interval is half-open. A task that ends at T and another that starts at T are both allowed. The exclusion constraint ignores cancelled and failed. Two transactions cannot keep overlapping busy tasks on the same satellite. A finished collection stays done, so a later search of that same past second does not offer it again.

`client_ref` is the caller's label. Tasking puts the order id here. It is not a foreign key. Sending the same `client_ref` again returns this row instead of creating a second task. The plane, class, duration, incidence, look side, pass, and squint are the collection the constellation chose. Lat, lon, mode, the requested look and pass, the incidence choice, and the acquisition window are what the caller asked for, so a portal that did not place the order can still show the task.

This is the Lisbon Spot task. It is an example, not a row the migration inserts. Placed after 2026-10-29T06:07:01Z, its status is already done.

| column | example |
| --- | --- |
| id | c0ffee00c0ffee00 |
| status | done |
| satellite | Aether-A1 |
| plane | Dawn |
| class | Standard |
| lat | 38.7223 |
| lon | -9.1393 |
| mode | SPOTLIGHT |
| requested_look | ANY |
| requested_pass | ANY |
| incidence_choice | performant |
| window_start | 2026-10-29 00:00:00+00 |
| window_end | 2026-10-30 00:00:00+00 |
| imaging_start | 2026-10-29 06:06:51+00 |
| imaging_end | 2026-10-29 06:07:01+00 |
| duration_s | 10 |
| incidence | 26.12 |
| look_side | RIGHT |
| pass | ASCENDING |
| squint | 0.400 |
| client_ref | a1b2c3d4e5f60718 |
| created_at | 2026-10-31 12:00:00+00 |
| cancelled_at | null |

## Tasking

### orders

One row is one order. `satellite`, `plane`, `class`, and `mode` are text copied at booking time. `task_id` is the constellation task for this order. `status` is the last status that task reported. There is no foreign key to the constellation catalog, and there is no overlap constraint here. The satellite calendar lives in `tasks`.

Cancelling an order asks the constellation to cancel the task, then stores the status it returns. The order row stays. A task in downlink, processing, done, or failed cannot be cancelled. Once a second, tasking asks the constellation for every order still scheduled, imaging, downlink, or processing and writes those statuses back. An update only lands while the order is still in one of those states, so a cancel that arrived first is not overwritten.

This row is the shape of the Lisbon Spot example. It is an example, not a row the migration inserts. A new tasking database has no orders until someone places one.

| column | example |
| --- | --- |
| id | a1b2c3d4e5f60718 |
| task_id | c0ffee00c0ffee00 |
| status | done |
| created_at | 2026-10-31 12:00:00+00 |
| cancelled_at | null |
| satellite | Aether-A1 |
| plane | Dawn |
| class | Standard |
| lat | 38.7223 |
| lon | -9.1393 |
| mode | SPOTLIGHT |
| requested_look | ANY |
| requested_pass | ANY |
| incidence_choice | performant |
| window_start | 2026-10-29 00:00:00+00 |
| window_end | 2026-10-30 00:00:00+00 |
| imaging_start | 2026-10-29 06:06:51+00 |
| imaging_end | 2026-10-29 06:07:01+00 |
| duration_s | 10 |
| incidence | 26.12 |
| look_side | RIGHT |
| pass | ASCENDING |
| squint | 0.400 |

`look_side` and `pass` are what the search computed. `requested_look` and `requested_pass` are what the user asked for. `incidence` is the angle of that collection. `incidence_choice` is which band they asked for, performant or widest.
