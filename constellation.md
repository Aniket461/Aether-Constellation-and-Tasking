# Aether constellation

Aether is a 12-satellite synthetic-aperture-radar constellation in low Earth orbit. Four sun-synchronous planes, three satellites in each plane, share the sky. Class, look side, and imaging mode decide which satellites can take a given scene.

This file is both the lesson and the specification. The first part explains every term the constellation uses, and what that term changes when a user asks for an image. The tables are the values the software will seed. The TLE epoch stays 2026-10-29 00:00:00 UTC. The app clock is wall-clock UTC. Every TLE is built from the elements here, and SGP4 is the definition of where each satellite is.

Scene size, duration, incidence, and squint for the imaging modes are the ICEYE tasking parameters.

## The shape of a task

A user clicks a point on the globe and asks whether any Aether satellite can take a SAR image of it inside a time window. The answer is either the earliest legal start, with the satellite and the geometry of that collection, or not feasible. If the user orders it, that satellite is busy for the length of the collection. The search window itself is not reserved.

Four different clocks show up in that sentence:

| Word | What it is | How long it is here |
| --- | --- | --- |
| Epoch | The instant the orbital elements are defined | 2026-10-29 00:00:00 UTC |
| Acquisition window | The span the user will accept for the image | Whatever they type, searched second by second |
| Imaging interval | The seconds the radar is actually on | 10–42 s for a nominal scene |
| Reserved slot | The imaging interval, stored on that satellite | Same as the imaging interval |

Widening the acquisition window can only add candidates. It does not make the satellite busy for longer. Ordering a 10-second Spot scene blocks that satellite for those 10 seconds, and a later request may use it the moment the interval ends.

## Synthetic aperture radar

A SAR satellite takes a picture with radio waves. It sends a short pulse at the ground and listens for the echo that bounces back. The satellite carries its own radio “light,” so the picture works at night. Clouds let these radio waves through, so a cloudy day is still a usable pass. A picture at 02:30 local time is a normal SAR picture.

The antenna on the satellite is physically short. While the satellite flies, it keeps sending pulses and keeps the echoes. Stacking those echoes is like having used one very long antenna. That pretend long antenna is the synthetic aperture. It is what makes the picture sharp along the direction of flight.

The longer the radar stares at one spot, the more echoes it can stack, and the more detail it can keep along the flight path. A dwell stares for 25 seconds. An ordinary spot stares for 10.

The radio still has to be aimed. The beam has to hit the place from an allowed angle, from a side the antenna can point, while the satellite is flying the direction the user asked for. The rest of this document is those aiming rules.

Two things in the picture come straight from that aim, and they are why the allowed angle has a floor and a ceiling:

- Layover. From a steep aim, a tall building or a mountain face gets squeezed toward the satellite. The roof shows up in the wrong place, closer to the satellite than the base of the building.
- Radar shadow. From a shallow aim, a tall object leaves a dark patch behind it, on the side the radio cannot reach. The satellite casts that shadow. The sun is not involved.

Speckle is the salt-and-pepper grain in a SAR picture. A flat field is made of many tiny surfaces. Their echoes add and cancel, so one small square of the picture comes out bright and the next one dark, even though the ground is the same. A look is one independent view of that square. Averaging several looks smooths the grain. Modes with more looks look cleaner. They usually stare longer, or they give up some sharpness, to collect those extra views.

Polarization is the direction the radio wave is “standing” as it travels. VV means the satellite sends the wave standing up-and-down, in the plane that contains straight-up and the line of sight, and it listens for the echo that comes back standing the same way. Every Aether mode is VV. A user cannot ask for a sideways wave (HH) or a mixed one (VH or HV). There is nothing to trade against picture size or sharpness.

## Low Earth orbit

Aether flies between 505 km and 560 km. A satellite at that height completes one revolution in about 95 minutes and moves at roughly 7.6 km/s relative to the Earth’s center. The path of the point directly underneath the satellite is the ground track. Imaging does not happen on that line. It happens beside it, tens to hundreds of kilometers off to the left or the right.

Altitude changes four things at once:

- Period and mean motion. Lower flies faster around the Earth. The dusk plane at 505 km completes 15.20 revolutions per day. The afternoon plane at 560 km completes 15.02. Over one day the lower satellite gains a noticeable fraction of a revolution on the higher one, so the pattern of which satellite arrives first keeps changing.
- Reach of a given incidence angle. A higher satellite sees the same incidence angle farther from the ground track. The distances are listed under Incidence.
- Sun-synchronous inclination. A higher orbit needs a slightly larger inclination to keep the same local time of day. The values are in the plane table.
- Drag. Thinner air at higher altitude slows the decay. Aether uses one small drag term on every satellite, so this simulator does not give the 505 km plane a faster decay than the 560 km plane. The drag term is there so the orbit model is complete.

The highest latitude the ground track reaches is `180° − inclination`, about 82.4° to 82.6° depending on the plane. A point closer to the pole than that never has a satellite pass overhead. A very wide look can still see a little past the apex, and the incidence test is what decides it.

## Orbital elements

These numbers, at one epoch, are enough for SGP4 to produce position and velocity at any later or earlier time. The elements are mean elements: they describe a smoothed orbit. SGP4 adds the periodic wobble from Earth’s shape back on top. A latitude computed by hand from the elements will sit close to the propagated latitude, not necessarily on top of it.

### Epoch

The epoch is the one instant at which the orbital elements are the mean state of the satellite. Aether’s epoch is 2026-10-29 00:00:00 UTC. Inclination, RAAN, eccentricity, argument of perigee, mean anomaly, and mean motion are all stated at that second. The epoch stays fixed. The globe, the search, and the orders use wall-clock UTC, not a clock that starts at the epoch.

An element set has no meaning without its epoch. Mean anomaly 0° means “at perigee,” and the epoch says when that happens. Move the epoch to an hour later and leave mean anomaly at 0°, and the satellite reaches perigee an hour later. Leave the epoch alone and add 227.2984° to the mean anomaly, and you have described the same satellite one hour later, because Aether-A1’s mean motion carries it 227.2984° of mean anomaly every hour (`15.15322385 rev/day × 360° / 24`).

SGP4 does not store the epoch as a calendar date. It stores a year and a day-of-year with a fraction. 2026 is not a leap year, so the day count is:

| Month | Days |
| --- | --- |
| January | 31 |
| February | 28 |
| March | 31 |
| April | 30 |
| May | 31 |
| June | 30 |
| July | 31 |
| August | 31 |
| September | 30 |
| Sum through September | 273 |
| 29 October | day 302 |

The fraction is the time of day divided by 24 hours. Midnight is exactly 0, so the TLE field is `302.00000000` and the year field is `26`. A different time of day on the same date would be a different epoch:

| UTC on 29 October 2026 | Day fraction | TLE epoch field |
| --- | --- | --- |
| 00:00:00 | 0 / 24 = 0 | `26302.00000000` |
| 06:00:00 | 6 / 24 = 0.25 | `26302.25000000` |
| 12:30:00 | 12.5 / 24 = 0.52083333 | `26302.52083333` |
| 18:00:00 | 18 / 24 = 0.75 | `26302.75000000` |

The leading `26` is the year. The rest is the day. There is no month and no clock in the line. `26302.25` is 06:00 UTC on day 302 of 2026, which is 29 October.

Internally SGP4 turns that field into a Julian date. The Julian date of this epoch is 2461342.5, which is the continuous day count used by the propagator. Minutes from the epoch are `(Julian date of the question − 2461342.5) × 1440`. Propagating to the epoch itself is a propagation of 0 minutes, and the state that comes back is the mean elements plus the short-period wobble SGP4 always adds.

Days away from the epoch, the drag term slowly lowers the orbit, the period shortens, and the local time of the node can drift off the value designed for the original altitude. Tasking near 29 October 2026 stays on the altitudes and local times in the plane table.

### Semi-major axis, altitude, perigee, and apogee

The semi-major axis is half the long axis of the orbit ellipse, measured from the center of the Earth. Altitude in the plane table is the height of that axis above a spherical Earth of radius 6378.137 km, so the axis is `6378.137 + altitude`.

Eccentricity is 0.001 on every satellite. A circle has eccentricity 0. At 0.001 the orbit is 7 km lower at perigee and 7 km higher at apogee than the plane altitude. Perigee is the closest point to Earth. Apogee is the farthest.

That 14 km of altitude variation moves the incidence geometry by a few kilometers of ground range and changes speed slightly around the orbit. It is large enough to be a real ellipse in the TLE, and small enough that the plane altitude remains the right mental model.

### Argument of perigee

Argument of perigee is the angle, measured in the orbital plane in the direction of flight, from the ascending node to perigee. Aether uses 90° on every satellite. Perigee therefore sits about a quarter of a revolution after the northbound equator crossing, which is the northern extreme of the orbit. Apogee sits over the southern extreme.

The northern pass of a given satellite is the low, slightly faster pass. The southern pass is the high, slightly slower one. Because every satellite shares 90°, this north-low / south-high pattern is the same on all twelve.

### Inclination

Inclination is the tilt of the orbital plane relative to the equator. 0° would be an equatorial orbit. 90° would be a true polar orbit. Aether flies at about 97.4° to 97.6°, which is retrograde: the satellite’s inertial motion has a westward component. The ground track still runs north and south.

The tilt is set to the sun-synchronous value for each altitude, not to 90°, so that the plane keeps a fixed relationship with the sun. The mechanism is Earth’s equatorial bulge, described by the zonal coefficient J2. The bulge tugs a retrograde orbit so that the node slides east by about one degree per day, which matches the sun’s apparent motion. A higher orbit is tugged less, so it needs a little more inclination to get the same daily slide. That is why 560 km is at 97.6316° and 505 km is at 97.4207°.

### Right ascension of the ascending node

The ascending node is the point where the satellite crosses the equator going north. Right ascension of the ascending node, RAAN, is the angle of that point measured in the equator from the vernal equinox, the fixed inertial reference direction, positive toward the east.

RAAN orients the plane in inertial space. It is not a clock time by itself. The same RAAN corresponds to a different local time of day on a different date, because the sun moves. At this epoch the sun’s right ascension is 213.2760°, and each plane’s RAAN was chosen from that sun angle and the desired local time:

`RAAN = solar right ascension + 15° × (LTAN hours − 12)`

Fifteen degrees is one hour of Earth rotation.

### Local time of the ascending node

LTAN is the local solar time at the longitude where the satellite crosses the equator going north. Because the planes are sun-synchronous, that clock time repeats from day to day.

The descending node, the southbound equator crossing, lies 180° of right ascension away, so its local solar time is 12 hours different.

| Plane | Ascending node | Descending node |
| --- | --- | --- |
| Dawn | 06:00 | 18:00 |
| Morning | 10:30 | 22:30 |
| Afternoon | 14:30 | 02:30 |
| Dusk | 18:00 | 06:00 |

LTAN is not the local time of the photograph. The photograph happens when the satellite reaches the point’s latitude, which is minutes after the equator crossing. On these orbits a satellite reaches 40° latitude about 11 minutes after the node, and 60° latitude about 16 minutes after the node. The local time of a mid-latitude image on the ascending pass sits near the LTAN, shifted by those minutes and by the change in longitude along the inclined track. The descending pass is the other row of the table.

A dawn-plane satellite therefore images near dawn on its northbound pass and near dusk on its southbound pass. The plane name is the ascending node only.

Dawn and dusk planes lie close to the terminator, the day-night boundary, so the satellite spends less of each orbit in Earth’s shadow. Morning and afternoon planes cut deeper into the night side and spend longer in eclipse. Eclipse changes battery and thermal load on a real bus. The radar supplies its own illumination, and the Aether scheduler does not reject a pass for eclipse.

### Mean anomaly

Mean anomaly is where the satellite is along the ellipse at the epoch, measured from perigee in the direction of flight. 0° is perigee. 180° would be apogee. It advances almost uniformly with time, which is why it is the parameter used to space satellites.

With argument of perigee at 90° and a very small eccentricity, the position in the orbit at the epoch is approximately:

| Mean anomaly | Where the satellite is |
| --- | --- |
| 0° | Near the northern extreme, at perigee |
| 120° | In the southern hemisphere, still heading south |
| 240° | Past the southern extreme, heading back north toward the equator |

The three satellites of one plane use 0°, 120°, and 240°. They are equally spaced around the ellipse. The next one reaches a given point in the orbit about a third of a period later, close to 32 minutes.

That follower does not fly the same ground track. In 32 minutes the Earth rotates about 8°, which is about 890 km at the equator and less by `cos(latitude)` as you move away from it. The follower is on a parallel track, hundreds of kilometers away. A Spot scene is 5 km wide. A Scan Wide scene is 200 km wide. Neither covers an 890 km gap. Phasing spreads the satellites in time. It does not, by itself, give three chances at the same point on one pass.

### Mean motion and period

Mean motion is how many revolutions the satellite completes in one day. Period is the time for one revolution. They are the same fact:

`mean motion (rev/day) = 1440 / period (minutes)`

SGP4 wants mean motion. A TLE stores it to eight decimal places, which is why the plane table looks over-precise. Those digits come from the semi-major axis and the gravitational parameter 398600.4418 km³/s².

Each revolution, the Earth rotates about 24° under the orbit, about 2650 km at the equator. The next ground track of the same satellite lies that far west of the previous one. A single satellite therefore does not see the same point on consecutive revolutions unless the swath is thousands of kilometers wide, which none of these modes are. Opportunities come when some satellite’s track passes within the off-track distance of the requested incidence band, on a later revolution or from another plane.

### Drag term B*

B* is the drag coefficient carried in the TLE, in units of 1 / Earth radii. SGP4 uses it to pull the orbit slowly inward. Aether sets every satellite to `1.0 × 10⁻⁵`. The value is small and shared, so drag does not become a difference between classes or planes. Near the epoch the design altitudes hold.

### Catalog number

Each satellite has a catalog number from 95001 to 95012. It is the integer written into the TLE in the slot a NORAD catalog number would occupy. It identifies the satellite in the element set. It is not a radio frequency, an orbit, or a mode.

### TLE

A two-line element set is those mean elements written as two fixed-width lines of 69 characters, plus an optional name line for people. SGP4 reads line 1 and line 2. The name is not part of the math. The globe and the feasibility check both propagate these lines, so the dot on the map and the offered imaging time are the same orbit.

This is Aether-A1, the whole set:

```text
AETHER-A1
1 95001U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    16
2 95001  97.4778 123.2760 0010000  90.0000   0.0000 15.15322385    16
```

Every column is spoken for. A space is a space on purpose, and a missing digit is a different satellite. The last character of each line is a checksum, not part of the element in front of it. On Aether-A2’s second line the last two characters are `10`. That is revolution number `1` and checksum `0`, not revolution 10.

Line 1 carries identity, the epoch, and drag. Line 2 carries the orbit and where the satellite is on it.

Line 1 of Aether-A1, column by column. Columns are numbered from 1.

| Columns | Width | Field | Value on Aether-A1 | What it means |
| --- | --- | --- | --- | --- |
| 1 | 1 | Line number | `1` | Always 1 on the first line |
| 2 | 1 | Spacer | space | |
| 3–7 | 5 | Catalog number | `95001` | This satellite |
| 8 | 1 | Classification | `U` | Unclassified. `C` and `S` exist in the format and are unused here |
| 9 | 1 | Spacer | space | |
| 10–17 | 8 | International designator | `26001A` | Year `26`, launch `001` of that year, piece `A`. A label we assign so the line is legal. Two trailing spaces pad the field to 8 |
| 18 | 1 | Spacer | space | |
| 19–20 | 2 | Epoch year | `26` | 2026 |
| 21–32 | 12 | Epoch day | `302.00000000` | Day 302 at midnight, from the table in Epoch |
| 33 | 1 | Spacer | space | |
| 34–43 | 10 | First derivative of mean motion, divided by 2 | ` .00000000` | Zero. A leading space is the plus sign. The stored number is half the derivative, in rev/day² |
| 44 | 1 | Spacer | space | |
| 45–52 | 8 | Second derivative of mean motion, divided by 6 | ` 00000-0` | Zero. The decimal point is assumed after the sign, so this is `0.00000 × 10⁰` |
| 53 | 1 | Spacer | space | |
| 54–61 | 8 | B* | ` 10000-4` | `0.10000 × 10⁻⁴` = `1.0 × 10⁻⁵` per Earth radius. Decimal point assumed |
| 62 | 1 | Spacer | space | |
| 63 | 1 | Ephemeris type | `0` | The standard SGP4 model. Another digit would select a different propagator |
| 64 | 1 | Spacer | space | |
| 65–68 | 4 | Element set number | `1` | First publication of this set. A later update would be 2, 3, … |
| 69 | 1 | Checksum | `6` | See below |

The two derivative fields are zero, so the line does not impose an extra secular drift. The drag SGP4 applies comes from B*.

The assumed-decimal fields are the part that does not look like an ordinary number. ` 10000-4` is read as sign `+`, mantissa `0.10000`, exponent `−4`. A real decaying object often has a B* more like ` 14593-3`, which is `0.14593 × 10⁻³`. The minus before the exponent counts in the checksum. Letters, spaces, plus signs, and decimal points do not.

Line 2 of Aether-A1:

| Columns | Width | Field | Value on Aether-A1 | What it means |
| --- | --- | --- | --- | --- |
| 1 | 1 | Line number | `2` | Always 2 |
| 2 | 1 | Spacer | space | |
| 3–7 | 5 | Catalog number | `95001` | Must match line 1 |
| 8 | 1 | Spacer | space | |
| 9–16 | 8 | Inclination | ` 97.4778` | Degrees. The leading space is padding so the field is 8 wide |
| 17 | 1 | Spacer | space | |
| 18–25 | 8 | RAAN | `123.2760` | Degrees from the vernal equinox |
| 26 | 1 | Spacer | space | |
| 27–33 | 7 | Eccentricity | `0010000` | `0.0010000`. The decimal point is assumed at the front and is not written |
| 34 | 1 | Spacer | space | |
| 35–42 | 8 | Argument of perigee | ` 90.0000` | Degrees |
| 43 | 1 | Spacer | space | |
| 44–51 | 8 | Mean anomaly | `  0.0000` | Degrees. Two leading spaces, then `0.0000` |
| 52 | 1 | Spacer | space | |
| 53–63 | 11 | Mean motion | `15.15322385` | Revolutions per day |
| 64–68 | 5 | Revolution number at epoch | `    1` | Which revolution this epoch falls in. It does not move the satellite. SGP4 does not use it for the state |
| 69 | 1 | Checksum | `6` | |

The checksum is the only field that is derived. Take the first 68 characters. Add the integer value of every digit. Add 1 for every minus sign. Ignore everything else. The units digit of that sum is column 69.

On Aether-A1’s line 1 the digits add to 44 and there are two minus signs (the exponents of the second derivative and of B*), so the total is 46. The units digit is 6, which is what the line ends with. On line 2 the digits add to 126 and there is no minus sign. The units digit is 6 again. A line whose checksum does not match is rejected. That is how a shifted column gets caught.

The twelve sets, one per satellite. Only line 2 changes inside a plane, and only in the mean anomaly, until the next plane changes inclination, RAAN, and mean motion too.

```text
AETHER-A1
1 95001U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    16
2 95001  97.4778 123.2760 0010000  90.0000   0.0000 15.15322385    16

AETHER-A2
1 95002U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    17
2 95002  97.4778 123.2760 0010000  90.0000 120.0000 15.15322385    10

AETHER-A3
1 95003U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    18
2 95003  97.4778 123.2760 0010000  90.0000 240.0000 15.15322385    14

AETHER-B1
1 95004U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    19
2 95004  97.5544 190.7760 0010000  90.0000   0.0000 15.08756051    13

AETHER-B2
1 95005U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    10
2 95005  97.5544 190.7760 0010000  90.0000 120.0000 15.08756051    17

AETHER-B3
1 95006U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    11
2 95006  97.5544 190.7760 0010000  90.0000 240.0000 15.08756051    11

AETHER-C1
1 95007U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    12
2 95007  97.6316 250.7760 0010000  90.0000   0.0000 15.02237003    16

AETHER-C2
1 95008U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    13
2 95008  97.6316 250.7760 0010000  90.0000 120.0000 15.02237003    10

AETHER-C3
1 95009U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    14
2 95009  97.6316 250.7760 0010000  90.0000 240.0000 15.02237003    14

AETHER-D1
1 95010U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    16
2 95010  97.4207 303.2760 0010000  90.0000   0.0000 15.20278457    19

AETHER-D2
1 95011U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    17
2 95011  97.4207 303.2760 0010000  90.0000 120.0000 15.20278457    13

AETHER-D3
1 95012U 26001A   26302.00000000  .00000000  00000-0  10000-4 0    18
2 95012  97.4207 303.2760 0010000  90.0000 240.0000 15.20278457    17
```

Read Aether-A2’s line 2 against Aether-A1’s. Catalog, mean anomaly, mean-motion digits, and both checksums change. Inclination and RAAN stay, because they share the dawn plane. Read Aether-B1 against Aether-A1 and the inclination, RAAN, and mean motion all change, because the plane changed.

### SGP4

SGP4 is the propagator that consumes a TLE. Given line 1, line 2, and a UTC time, it returns position and velocity in the Earth-centered inertial frame, kilometers and kilometers per second, with the axes fixed to the stars. The X axis points at the vernal equinox. The Z axis points at the north celestial pole. The Y axis completes a right-handed set.

The globe and the latitude of a pass live in the Earth-centered Earth-fixed frame, which rotates with the planet. The angle between those two frames at a given UTC is Greenwich mean sidereal time, GMST: the angle, measured in the equator, from the vernal equinox eastward to the Greenwich meridian. At the Aether epoch, GMST is 37.3407°, which is 2.4894 hours of sidereal time. A right ascension converts to longitude by subtracting GMST. The dawn plane’s ascending node, at RAAN 123.2760°, sits at longitude `123.2760 − 37.3407 = 85.9353°` east at that instant. Aether-A1 is not on that node at the epoch. Its mean anomaly is 0° and its argument of perigee is 90°, so it is near the northern extreme of the orbit while the node itself is an empty point on the equator.

From the fixed-frame state the scheduler computes geodetic latitude, longitude, and altitude, then the look angles in the next section. The browser never runs SGP4. It draws states the server has already computed.

## Every angle

An angle is meaningless until you know three things: where the vertex is, which direction is zero, and which way is positive. The orbit uses one family of angles. The radar look uses another. They share the word “azimuth” for two different measurements, and they share nothing else.

### Where the satellite is in its orbit

Picture the dawn orbit face-on, argument of perigee 90°, flying from the ascending node toward the north:

```text
                         perigee
                    northern extreme
                          *
                       /     \
                     /         \
     ascending *----< equator >----* descending
     node        \               /        node
                   \           /
                     \       /
                       \   /
                         *
                       apogee
                    southern extreme
```

The satellite moves from the ascending node, up the left side of that sketch, through perigee at the top, down toward the descending node, through apogee at the bottom, and back to the ascending node.

| Angle | Vertex | Zero direction | Positive sense | Aether |
| --- | --- | --- | --- | --- |
| Inclination | Earth’s center | Equatorial plane | Toward the north pole of the orbit | 97.42° to 97.63° by plane |
| RAAN | Earth’s center | Vernal equinox, in the equator | East | 123.2760° dawn, then 190.7760°, 250.7760°, 303.2760° |
| Argument of perigee | Earth’s center | Ascending node, in the orbital plane | Direction of flight | 90° on every satellite |
| True anomaly | Earth’s center, from the focus | Perigee | Direction of flight | The real angle along the ellipse. At the epoch it is within a fraction of a degree of the mean anomaly, because eccentricity is 0.001 |
| Eccentric anomaly | Center of the ellipse, not the focus | Perigee | Direction of flight | The auxiliary angle used to solve the ellipse. Equal to the true and mean anomalies only on a circle |
| Mean anomaly | Along the orbit | Perigee | Direction of flight, uniformly in time | 0°, 120°, or 240° at the epoch |
| Argument of latitude | Earth’s center | Ascending node | Direction of flight | Argument of perigee plus true anomaly. At the epoch, about 90°, 210°, or 330° |

Inclination is the hinge of the plane. Raising it moves the highest latitude of the ground track. Past 90° the orbit is retrograde, which is what lets the node precess east and hold a sun-synchronous local time. RAAN is the hinge’s compass direction in inertial space. Argument of perigee rotates the ellipse inside the plane so that perigee, the low point, lies at a chosen latitude. Mean anomaly slides the satellite along that ellipse. True anomaly is the same slide measured as a geometric angle from the focus, and it is the one that enters the argument of latitude. With our eccentricity the difference between true and mean anomaly is about `2e` radians at most, under 0.2°. Using mean anomaly as a picture of “where along the orbit” is safe. Using it as a substitute for true anomaly inside a precise latitude calculation is not, which is why the latitude of a pass comes from SGP4 and not from the raw mean anomaly.

Argument of latitude is the angle operators actually point at when they say “thirty degrees past the node.” It is zero at the ascending node, 90° at the northern extreme, 180° at the descending node, 270° at the southern extreme. Combined with inclination, it fixes the latitude:

`sin(latitude) = sin(inclination) × sin(argument of latitude)`

On the dawn plane, inclination 97.4778°, an argument of latitude of 40.41° or 139.59° is latitude 40°. That point is 10.67 minutes of flight from the ascending node at the dawn period of 95.03 minutes. The matching southern latitudes sit past the descending node.

Longitude of the sub-satellite point is a different angle. It is the geographic longitude of nadir, measured east from Greenwich, and it depends on RAAN, argument of latitude, inclination, and GMST at that UTC. It advances through about 360° every revolution in inertial space and, on the ground, walks west by about 24° per revolution because the Earth has turned the other way underneath.

The flight-path angle is the angle between the velocity and the local horizontal at the satellite. On a circle it is 0°. On this ellipse it is a fraction of a degree, uphill on the way from perigee to apogee and downhill on the way back. It is small enough to ignore when picturing a pass, and it is already inside SGP4’s velocity vector when the scheduler computes look side.

### Where the beam hits the ground

Stand the same satellite above a spherical Earth and look sideways at one range direction. Nadir goes to the center of the Earth. The point being imaged is off to one side.

```text
                         satellite
                            *
                           /|
                          / |
                 slant   /  |  off-nadir, at the satellite
                        /   |
                       /    | nadir
                      *-----*
                    point   nadir point
                      \
                       \ incidence, at the point
                        \
                         local vertical, straight up
```

The horizontal at the point is perpendicular to that local vertical. Elevation and grazing angle are measured up from the horizontal. Incidence is measured out from the vertical. They are complements at the same vertex: elevation = grazing angle = `90° − incidence`.

| Angle | Vertex | Zero direction | What a larger number means |
| --- | --- | --- | --- |
| Off-nadir, also called the look angle | The satellite | Straight down, toward Earth’s center | The antenna is tipped farther off vertical |
| Earth-central angle | Earth’s center | The nadir direction | The point is farther from the ground track, as an angle |
| Incidence | The point on the ground | Local vertical, straight up | A shallower radar look, a longer radar shadow, a finer ground-range resolution |
| Elevation | The point on the ground | Local horizontal | The satellite stands higher in the sky |
| Grazing angle | The point on the ground | Local horizontal | The same number as elevation. SAR writing uses this name for the complement of incidence |
| Squint | The satellite | Broadside, perpendicular to the velocity | The beam looks forward or backward along the track instead of exactly abeam |
| Observer azimuth | The point on the ground | True north, in the local horizontal | The compass bearing from the point to the satellite. 90° is east |
| SAR azimuth | Along the flight direction | The start of the scene, along track | Position inside the image in the direction of motion. This is not a compass bearing |

Incidence, off-nadir, and the earth-central angle are one triangle. On a sphere:

`incidence = off-nadir + earth-central angle`

`sin(off-nadir) = (Earth radius / (Earth radius + altitude)) × sin(incidence)`

Earth radius in this model is 6378.137 km. The ground distance from the nadir point to the imaged point is `Earth radius × earth-central angle in radians`. The slant range is the length of the beam, from satellite to point.

A worked look, morning-plane altitude 540 km, incidence 30°:

| Quantity | Value |
| --- | --- |
| Incidence | 30° |
| Elevation and grazing angle | 60° |
| Off-nadir | 27.4499° |
| Earth-central angle | 2.5501° |
| Distance from the ground track | 283.88 km |
| Slant range | 615.63 km |
| Check | 27.4499° + 2.5501° = 30° |

Read the elevation before reading the word “shallow.” At 30° incidence the satellite is 60° above the horizon. It is high in the sky, looking somewhat sideways. The performant Spot band, incidence 20–40°, is elevation 70° down to 50°. The requestable edge at 45° incidence is a satellite 45° above the horizon. The steep edge at 10° incidence is a satellite 80° above the horizon, 10° off zenith. Aether’s “shallow” looks are still the upper half of the sky. What gets worse out at 40–45° is the radar shadow and the obliqueness of the image, while ground-range resolution improves because `sin(incidence)` is larger.

Off-nadir is the number the antenna has to be steered by. For that 30° incidence the bus tips 27.45° off nadir, not 30°. Bus reach is specified in incidence, the quantity the image cares about. The steering law converts it to off-nadir at the current altitude.

Squint is the other steering axis, drawn looking down on the flight path:

```text
                 forward, positive squint
                          ^
                          |
                          |
     left look  <---------+-------->  right look
                          |
                          |  velocity
                          v
                 backward, negative squint

                 broadside is the left-right line, squint 0
```

A squint of +20° aims the beam 20° forward of broadside. The point being imaged is then ahead of the satellite, and the satellite can start the collection before it draws abreast. Spot and the dwell modes are built to squint up to 45° either side of broadside. The scheduler still aims for squint 0. Because it steps in whole seconds, the rule is the one in Finding the earliest time: the second closest to the sign change, accepted when that squint is within 1°.

Observer azimuth answers “which compass direction is the satellite from the point?” It changes continuously through a pass: the satellite rises in one bearing and sets in another. SAR azimuth answers “where along the image is this pixel?” A 50 km Strip scene has about 50 km of SAR azimuth. Swapping the two words is the usual way to misread a requirement. A task that says “azimuth 90°” without saying which one is unfinished.

### What changing an angle does

| Change | Effect on the constellation |
| --- | --- |
| Inclination up by 0.1° | The ground track reaches about 0.1° less latitude, because these orbits are retrograde and the apex is `180° − inclination`. The sun-synchronous local time also slips, because the inclination was set for one nodal rate |
| RAAN up by 15° | Local time of both nodes moves one hour later |
| Argument of perigee away from 90° | Perigee leaves the northern extreme. The low, fast pass moves to another latitude |
| Mean anomaly up by 120° | The satellite swaps places with the next one in its plane. Same orbit, different clock |
| A tighter incidence band | The legal ring around the ground track gets thinner. Scan Wide at 21–26° is about 50 km of ground range thick. Spot at 20–40° is more than 200 km thick |
| Look side from `any` to `right` | Every left-hand geometry disappears, including the left-hand half of every either-side satellite |
| Squint held at 0 | The collection can start only at the broadside instant, even on a dwell mode that could have started earlier |

## Planes

### In plain words

A plane is the loop a satellite keeps flying around the Earth. Picture a hoop tipped almost from pole to pole. Every satellite on that hoop follows the same loop. Aether has four hoops, and three satellites ride each one, spread evenly around it. The next satellite on the same hoop comes by about 32 minutes later.

The hoop is set so that it stays lined up with the sun. A satellite on a given hoop therefore crosses the equator going north at the same local time every day. That clock time is the name of the plane.

The name is only the northbound crossing. Half an orbit later the same satellite crosses the equator going south, and the local time is 12 hours away. A dawn satellite is a morning satellite on the way north and an evening satellite on the way south.

| Plane | Going north | Going south | What that means |
| --- | --- | --- | --- |
| Dawn | about 06:00 | about 18:00 | Northbound pictures near sunrise. Southbound pictures near sunset |
| Morning | about 10:30 | about 22:30 | Northbound pictures in the late morning. Southbound pictures late at night |
| Afternoon | about 14:30 | about 02:30 | Northbound pictures in the mid-afternoon. Southbound pictures after midnight |
| Dusk | about 18:00 | about 06:00 | Northbound pictures near sunset. Southbound pictures near sunrise |

On the globe each hoop has its own color. Dawn is blue, Morning is gold, Afternoon is coral, and Dusk is green. The three satellites of one color sit on one ring.

The plane picks the time of day of the picture. The radar makes its own radio light, so a night pass is still a good picture. The name does not say whether the ground is sunny. It says when the satellite is overhead.

A plane is one orbit orientation: one inclination, one RAAN, one altitude. Three satellites share it, spaced in mean anomaly. Four planes give four local times of day and four ground-track families. These planes, the bus classes, and the twelve satellites are rows inserted when the API applies its migrations. The running constellation is that database, not a list compiled into the program.

| Plane | LTAN | Altitude | Inclination | RAAN | Period | Mean motion | Track reaches |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Dawn | 06:00 | 520 km | 97.4778° | 123.2760° | 95.03 min | 15.15322385 rev/day | 82.52° latitude |
| Morning | 10:30 | 540 km | 97.5544° | 190.7760° | 95.44 min | 15.08756051 rev/day | 82.45° latitude |
| Afternoon | 14:30 | 560 km | 97.6316° | 250.7760° | 95.86 min | 15.02237003 rev/day | 82.37° latitude |
| Dusk | 18:00 | 505 km | 97.4207° | 303.2760° | 94.72 min | 15.20278457 rev/day | 82.58° latitude |

Choosing a plane, by asking for a pass direction and then seeing which satellite wins, chooses the local time of the collection and the track family. A mode that exists on only some planes cannot be given an earliest time from a plane that lacks it. Spot Fine has no satellite in the afternoon plane. A Spot Fine search can still succeed on dawn, morning, or dusk, and the earliest of those is the offer.

## Looking at the ground

The definitions, the triangle, and a fully worked 30° look are in Every angle. Here is what those angles do to a task.

### Ground track and nadir

The ground track is the curve of nadir points. Nadir is the direction from the satellite toward the center of the Earth, and the nadir point is where that line meets the surface. Incidence there is 0°. Aether does not take images at nadir.

### Off-nadir and incidence

Off-nadir is the angle at the satellite, between the nadir direction and the line to the point. Incidence is the angle at the point, between the local vertical and the line to the satellite. Incidence is the number the user specifies. On a curved Earth it is larger than the off-nadir angle, because the local vertical is not parallel to the satellite’s nadir.

A larger incidence means a shallower look and a point farther from the track. Ground-range resolution, the across-track resolution once the slant measurement is laid on the ground, scales roughly as `slant resolution / sin(incidence)`. Moving from 20° to 40° raises `sin` from 0.34 to 0.64, so the same slant resolution becomes a finer ground resolution. The shallow look also lengthens radar shadows. The steep look, down near 10°, shortens shadows, worsens ground-range resolution, and increases layover. The performant band is the interval where the mode’s published resolution and ambiguity hold. The edges are a quality boundary, not a hard stop of the antenna, except where the bus reach or the mode’s requestable range says so.

Approximate distance from the ground track to the point, for the morning plane at 540 km and a spherical Earth:

| Incidence | Distance from the track |
| --- | --- |
| 10° | 88 km |
| 15° | 133 km |
| 20° | 180 km |
| 21° | 190 km |
| 26° | 241 km |
| 29° | 273 km |
| 33° | 318 km |
| 35° | 342 km |
| 40° | 407 km |
| 45° | 480 km |

The 505 km plane is about 5 to 30 km closer than this, and the 560 km plane about 5 to 15 km farther, depending on the angle. The scheduler does not use this table. It computes incidence from the propagated position. The table is the scale: a Spot performant request of 20–40° wants the track 180–410 km to one side of the point, and it refuses a track that passes 50 km away. Scan, at 21–29°, wants a thinner ring, about 190–270 km off the track. Scan Wide, at 21–26°, wants about 190–240 km. A thinner ring is why those modes more often come back not feasible.

There is a hole under the satellite. For a 20° minimum, anything inside about 180 km of the track is too steep. A satellite that flies directly over the point is the wrong satellite for that request, even though it is the closest one.

### Look side

Look side is which side of the velocity vector the point is on. Facing the direction of flight, the point is either left or right. A northbound satellite with the point to the east is looking right. The same point to the east of a southbound satellite is on the left, because the velocity has reversed. One geographic side of the track swaps its left/right name between the ascending and descending passes.

A fixed-look satellite can use only the side written on the roster. An either-side satellite can use both. `any` on the request accepts whichever side that satellite can already use. Setting `right` removes every left-only satellite, and it removes the either-side satellites on the half of their passes where the point lies to the left.

Look side and pass direction stack. `right` plus `ascending` asks for one specific combination: northbound, point on the right of the track. That is a much smaller set of seconds than `any` plus `any`.

### Pass direction

A pass is one journey of a satellite past the point. Ascending means latitude is increasing, which on these orbits is the northbound leg. Descending means latitude is decreasing, the southbound leg. The scheduler uses the sign of the latitude rate from SGP4. Near the apex the rate passes through zero as the satellite turns around; those seconds fall out of a request that names a single direction, because the rate is no longer clearly positive or negative.

`any` keeps both legs. Naming one leg keeps the local time that belongs to that leg on that plane, from the node table above.

### Squint and broadside

Broadside means the beam is perpendicular to the velocity, abeam of the satellite. Squint is the angle off that perpendicular in the along-track direction. Positive squint looks forward along the track. Negative squint looks backward. Squint 0 is broadside.

Spot and the dwell modes can squint by up to 45° either way. That lets the radar start illuminating a point before it is abeam and keep illuminating it afterward, which is how a long dwell collects 25 seconds of echoes on a 5 km scene. The Aether scheduler places every scene at squint 0. A second is legal when the point is broadside and the incidence, side, and pass direction are legal. The published squint range is part of the mode record for a later scheduler. Until then, dwell still occupies the satellite for 25 seconds, while the geometry test is the single broadside instant at the start.

Strip, Scan, and Scan Wide are broadside collections, so their squint is 0°. Spot Extended Area publishes no squint range. The scheduler treats it as broadside as well.

### Range and azimuth

Range is the across-track direction, the direction of the radar’s slant measurement. Azimuth is the along-track direction, the direction of flight. Scene sizes in this document are `range × azimuth`. Strip’s 30 × 50 km scene is 30 km wide across the track and 50 km long along the track.

A standard order centers that rectangle on the point of interest. The feasibility test checks incidence at the center. Across a wide scene the incidence actually changes from the near edge to the far edge. On a Scan Wide scene that change can be several degrees, so a center sitting inside 21–26° can have an edge outside it. The current rule accepts the scene when the center is legal.

## Imaging modes

A mode is a way of running the radar for one scene. It fixes the scene size, how long the radar is busy, the incidence band that meets the published quality, and how far the user is allowed to push that band.

Pulse bandwidth sets the slant-range resolution. A wider bandwidth separates targets that are closer together in range. Spot-class collections at 300 MHz of bandwidth are the 1 m ground-resolution family. Spot Fine uses 600 MHz and is the 0.5 m family. A Standard bus does not carry the 600 MHz mode, which is why Spot Fine is absent from that class.

Looks are the independent azimuth views averaged to reduce speckle. They are a property of how the mode is processed, not a field on the order.

The nominal duration is the reserved busy time for a scene of the nominal size, centered on the point. Strip, Scan, and Scan Wide can be tasked longer, up to the maximum duration, to extend the azimuth length. A standard Aether order does not offer that extension. The maximum is recorded so the mode table stays complete.

When the user leaves incidence blank, the request uses the performant band. The requestable band is the widest interval the constellation will accept for that mode. Scan and Scan Wide have no wider time-dominant band, so requestable and performant are the same. The other modes can be pushed out to 10–45°, and a collection outside the performant band is still scheduled. The published image quality applies inside the performant band.

| Mode | API value | Scene (range × azimuth) | Nominal | Maximum | Performant | Requestable | Squint |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Spot | `SPOTLIGHT` | 5 × 5 km | 10 s | 10 s | 20–40° | 10–45° | ±45° |
| Spot Fine | `SPOTLIGHT_FINE` | 5 × 5 km | 15 s | 15 s | 20–40° | 10–45° | ±45° |
| Spot Extended Area | `SPOTLIGHT_EXTENDED_AREA` | 15 × 15 km | 10 s | 10 s | 20–40° | 10–45° | — |
| Dwell | `SPOTLIGHT_EXTENDED_DWELL` | 5 × 5 km | 25 s | 25 s | 20–40° | 10–45° | ±45° |
| Dwell Fine | `SPOTLIGHT_EXTENDED_DWELL_FINE` | 5 × 5 km | 25 s | 25 s | 20–40° | 10–45° | ±45° |
| Dwell Precise | `SPOTLIGHT_EXTENDED_DWELL_PRECISE` | 5 × 5 km | 25 s | 25 s | 20–33° | 10–45° | ±45° |
| Strip | `STRIPMAP` | 30 × 50 km | 10 s | 70 s | 15–35° | 10–45° | 0° |
| Scan | `SCAN` | 100 × 100 km | 15 s | 70 s | 21–29° | 21–29° | 0° |
| Scan Wide | `SCAN_WIDE` | 200 × 300 km | 42 s | 84 s | 21–26° | 21–26° | 0° |

### Spot

The antenna is steered to keep one 5 × 5 km patch under illumination for 10 seconds, using one radar beam. Four azimuth looks, 1 m ground resolution. This is the default small-scene mode and the one the most satellites can fly. The track must pass 180–410 km to the side for a performant collection.

### Spot Fine

The same 5 × 5 km scene, held for 15 seconds, with the 600 MHz bandwidth and five looks, for 0.5 m ground resolution. Only Fine-class satellites carry it: Aether-A2, Aether-B3, and Aether-D1. The afternoon plane cannot serve this mode. A Spot Fine request that would have been trivial for Spot can move hours later, or become infeasible, because nine of the twelve satellites are out of the search.

### Spot Extended Area

Also called a sliding spot. The beam slides along a larger patch instead of staring at one point, covering 15 × 15 km in 10 seconds with two looks and 1 m ground resolution. The scene is the right choice when the target’s position is known only to within a few kilometers. It is on the same three Fine satellites as Spot Fine. The specification publishes no squint range for it.

### Dwell, Dwell Fine, and Dwell Precise

The satellite stares at a 5 × 5 km scene for 25 seconds. The extra time is more pulses and more azimuth looks: 12 looks for Dwell, 6 for Dwell Fine, 3 for Dwell Precise. Ground resolution is 1 m, 0.5 m, and 0.25 m. Information density goes up with the stare, which is what these modes are for: a small scene that will be examined closely.

All three fly on the Dwell-class satellites only: Aether-B2, Aether-C2, and Aether-D2. The dawn plane has no dwell satellite. Dwell Precise keeps its performant promise only from 20° to 33°. A user may still request up to 45°, and the bus itself stops at 40°, so the effective cap on that satellite is 40°.

Ordering a dwell blocks the satellite for 25 seconds rather than 10. A second point that becomes broadside 15 seconds later cannot share that satellite.

### Strip

One beam stays at a fixed broadside angle and the motion of the satellite paints a strip parallel to the track. The nominal scene is 30 km in range by 50 km in azimuth, 10 seconds, one look, 3 m ground resolution. The azimuth length can be extended, up to 500 km and 70 seconds, on a custom order this constellation does not offer yet.

Every Aether satellite can fly Strip. It is the mode with the most candidates. Its performant incidence, 15–35°, starts closer to the track than Spot, about 133 km off the track at the steep edge. A fixed-look satellite still has to have the point on its side.

### Scan

Scan uses four elevation beams, one after another, to paint adjacent strips in a single pass. The technique is a terrain-observation scan that also sweeps each beam forward to back along track so the illumination stays even. The result is a 100 × 100 km scene in 15 seconds at 15 m ground resolution, one look, and only an amplitude product. The price of the width is a narrow performant incidence band, 21–29°, a ring about 190–273 km off the track, and no wider requestable band.

Standard, Fine, and Wide satellites carry Scan. Dwell-class satellites do not. Nine satellites remain in the search, each of them still constrained to that thin ring and to its look side.

### Scan Wide

Eight elevation beams cover 200 km in range by 300 km in azimuth, in 42 seconds, at 27 m ground resolution. Performant and requestable incidence are both 21–26°, a ring only about 50 km thick, near 190–240 km off the track. The azimuth length can extend to 600 km and 84 seconds on a custom order that is not offered.

Only the three Wide satellites fly it, each locked to one side: Aether-A3 left on the dawn plane, Aether-C1 right on the afternoon plane, Aether-D3 left on the dusk plane. A left-looking Scan Wide request can be served from dawn or dusk. A right-looking Scan Wide request can be served only by Aether-C1, and only when that satellite’s track falls in the 21–26° ring during the window. This is the mode most likely to return not feasible, and the mode whose earliest time moves the most when another order already holds Aether-C1.

## Classes

A class is the bus and the antenna, not an orbit. Satellites of one class can share a plane with other classes. The class decides the mode list, whether the look is fixed, and the incidence interval the antenna can physically reach. A task is feasible for that satellite only when the mode is on its list, the point is on a side it can use, and the incidence is inside both the user’s range and the bus reach.

| Class | Look | Bus incidence | Modes |
| --- | --- | --- | --- |
| Standard | one side, fixed | 15–40° | Spot, Strip, Scan |
| Fine | either side | 15–45° | Spot, Spot Fine, Spot Extended Area, Strip, Scan |
| Dwell | either side | 15–40° | Spot, Dwell, Dwell Fine, Dwell Precise, Strip |
| Wide | one side, fixed | 15–35° | Strip, Scan, Scan Wide |

The bus reach cuts the mode’s requestable band. A Fine satellite can follow a Spot request out to 45°. A Standard satellite flying the same mode stops at 40°, because 40° is as far as that antenna is tasked. A Wide satellite flying Strip stops at 35° even though Strip’s requestable band goes to 45°. Scan’s own band is already inside every class that carries it.

Either-side satellites roughly double their geometric chances, because each pass has the point on exactly one side, and they can take that side. A fixed-look satellite misses every pass where the point is on the other side.

Standard is the earlier bus: one fixed look, and no 600 MHz mode. Fine adds the high-resolution and extended spot modes and can roll either way. Dwell gives the antenna time to the long stares and does not carry the multi-beam scan modes. Wide carries the broad swaths and looks to one side only.

## Roster

Catalog numbers run from 95001 to 95012. Mean anomaly is at the epoch. Orbit elements that are not in this table come from the satellite’s plane.

| Satellite | Catalog | Plane | Class | Look | Mean anomaly |
| --- | --- | --- | --- | --- | --- |
| Aether-A1 | 95001 | Dawn | Standard | right | 0° |
| Aether-A2 | 95002 | Dawn | Fine | either | 120° |
| Aether-A3 | 95003 | Dawn | Wide | left | 240° |
| Aether-B1 | 95004 | Morning | Standard | left | 0° |
| Aether-B2 | 95005 | Morning | Dwell | either | 120° |
| Aether-B3 | 95006 | Morning | Fine | either | 240° |
| Aether-C1 | 95007 | Afternoon | Wide | right | 0° |
| Aether-C2 | 95008 | Afternoon | Dwell | either | 120° |
| Aether-C3 | 95009 | Afternoon | Standard | right | 240° |
| Aether-D1 | 95010 | Dusk | Fine | either | 0° |
| Aether-D2 | 95011 | Dusk | Dwell | either | 120° |
| Aether-D3 | 95012 | Dusk | Wide | left | 240° |

At the epoch, every satellite with mean anomaly 0° is near its northern perigee, every 120° satellite is heading south in the southern hemisphere, and every 240° satellite is heading north toward the equator. A1, B1, C1, and D1 share that northern position in four different planes. They are not near each other over the Earth. Their planes are separated by hours of local time, which is tens of degrees of RAAN.

Who can fly what:

- Spot can be flown from every plane, by the Standard, Fine, and Dwell satellites. The three Wide satellites cannot.
- Strip can be flown by all twelve.
- Spot Fine and Spot Extended Area fly on Aether-A2, Aether-B3, and Aether-D1.
- Dwell, Dwell Fine, and Dwell Precise fly on Aether-B2, Aether-C2, and Aether-D2.
- Scan flies on every Standard, Fine, and Wide satellite.
- Scan Wide flies on Aether-A3 to the left, Aether-C1 to the right, and Aether-D3 to the left.

## What changes the earliest time

The scheduler walks every whole second of the acquisition window, for every satellite, and keeps the earliest second that passes every test below. Changing an input removes seconds, or removes whole satellites. The earliest survivor moves later, or the search comes back empty.

| Choice | What it removes |
| --- | --- |
| A narrower window | Every pass outside it, including passes that were geometrically perfect |
| A mode owned by fewer satellites | Whole planes. Spot Fine drops the afternoon plane and all Standard, Dwell, and Wide satellites |
| A tighter incidence band | Tracks that pass too close or too far. Scan Wide’s 5° band is a thin ring |
| `left` or `right` | The other side of every track, and every fixed-look satellite built for the opposite side |
| `ascending` or `descending` | One local time of day on each plane |
| An existing order | That satellite’s seconds inside the reserved imaging interval |

Leaving look side, pass direction, and incidence at their defaults leaves the largest set. The defaults are `any`, `any`, and the mode’s performant band.

## Finding the earliest time

The scheduler walks whole seconds, from the start of the acquisition window, over every satellite whose class lists the requested mode. SGP4 is called once per satellite per second. A second can start the scene only if every test below passes. The smallest such second in the constellation is the offer. If the walk finishes with none, the task is not feasible.

The Earth in this calculation is a sphere of radius 6378.137 km, the same sphere as the angle tables. SGP4 is run with the WGS84 gravity constant, because the mean motions in the TLEs were computed with μ = 398600.4418 km³/s². Position and velocity come back in SGP4’s inertial frame. The look tests use the Earth-fixed frame, so the inertial vectors are rotated by Greenwich sidereal time before any angle is measured.

### Tests on one second

**1. Class.** The mode must be on the satellite’s list. Spot drops Aether-A3, Aether-C1, and Aether-D3 before they are propagated. A fixed-look satellite fails later if the computed side is the other one. An either-side satellite can pass on either side.

**2. Point vector.** Latitude φ and longitude λ of the request, altitude 0:

`p = 6378.137 × (cos φ cos λ, cos φ sin λ, sin φ)` km

**3. Inertial state.** At UTC year, month, day, hour, minute, second:

`r_inertial, v_inertial = SGP4(line 1, line 2, that UTC)`

Both vectors are kilometers and kilometers per second.

**4. Rotate into the Earth-fixed frame.** θ is GMST in radians at that UTC.

`r_x = r_ix cos θ + r_iy sin θ`

`r_y = −r_ix sin θ + r_iy cos θ`

`r_z = r_iz`

The velocity is rotated the same way, then the rotation of the Earth is taken out so the vector is motion relative to the ground. ω = 7.2921150×10⁻⁵ rad/s.

`v_x = v_rot_x + ω r_y`

`v_y = v_rot_y − ω r_x`

`v_z = v_rot_z`

Latitude and longitude of the satellite, used for the pass test, come from that fixed-frame position:

`latitude = atan2(r_z, √(r_x² + r_y²))`

`longitude = atan2(r_y, r_x)`

`altitude = |r| − 6378.137`

**5. Incidence.** The line of sight from the point up to the satellite is `r − p`.

`cos i = (p · (r − p)) / (|p| |r − p|)`

`i = arccos(cos i)`

The second fails unless `i` sits in the intersection of the requested band, the mode’s requestable band, and the bus reach. A blank incidence field means the mode’s performant band.

**6. Look side.** Form the angular-momentum vector `h = r × v` with the Earth-fixed vectors. The sign of `h · p` is the side:

| `h · p` | Side of the ground track |
| --- | --- |
| negative | right |
| positive | left |

`any` accepts the computed side when the bus can look that way. `right` fails a positive triple product. A Standard or Wide satellite whose roster side disagrees with the sign fails even if the request said `any`.

**7. Pass direction.** The northward part of the Earth-fixed velocity, at the satellite’s own latitude φ and longitude λ:

`v_north = −sin φ cos λ · v_x − sin φ sin λ · v_y + cos φ · v_z`

Positive is ascending. Negative is descending. `any` accepts either sign.

**8. Squint.** Split the Earth-fixed velocity into the part perpendicular to `r`. That horizontal direction is along-track, `u_along`. The horizontal direction to the right of it is

`u_right = (−r / |r|) × u_along`

The vector from the satellite to the point is `p − r`.

`squint = atan2( (p − r) · u_along , |(p − r) · u_right| )`

Positive squint means the point is still ahead of the satellite. Negative means the satellite has passed broadside. Sampled once a second, squint rarely lands on 0. The broadside second is the one of the pair where the along-track dot product changes sign, chosen as whichever of those two seconds has the smaller absolute squint. It is accepted when that absolute value is at most 1°. One degree at a slant range of about 570 km is about 10 km along track, a little more than the satellite flies in one second.

**9. The collection fits.** The nominal duration of the mode is added to the second. That end must still be inside the acquisition window.

**10. The satellite is free.** The half-open interval `[start, start + duration)` must not overlap an active order on that satellite. A collection that ends at 06:07:01 and another that starts at 06:07:01 do not overlap.

### Worked request

Lisbon, one day from the epoch, Spot at the performant incidence, either side, either pass direction. No orders are on the books.

```json
{
  "pointOfInterest": { "lat": 38.7223, "lon": -9.1393 },
  "acquisitionWindow": {
    "start": "2026-10-29T00:00:00Z",
    "end": "2026-10-30T00:00:00Z"
  },
  "imagingMode": "SPOTLIGHT",
  "lookSide": "ANY",
  "passDirection": "ANY"
}
```

Spot’s performant band is 20–40°. The walk returns one winner:

```json
{
  "feasible": true,
  "satellite": "Aether-A1",
  "imagingTime": {
    "start": "2026-10-29T06:06:51Z",
    "end": "2026-10-29T06:07:01Z"
  },
  "durationSeconds": 10,
  "incidenceAngle": 26.12,
  "lookSide": "RIGHT",
  "passDirection": "ASCENDING"
}
```

Aether-A1 is the dawn-plane Standard satellite, catalog 95001, fixed to the right. Its TLE is the first set in this document. The seconds below are that satellite at 2026-10-29 06:06:51 UTC.

SGP4 returns the inertial state:

| | x | y | z |
| --- | --- | --- | --- |
| Position, km | −2503.667534 | 4794.241287 | 4269.926274 |
| Velocity, km/s | 3.276133 | −3.548343 | 5.879823 |

GMST at that UTC is 2.25678608 rad (129.304318°). cos θ = −0.63343919, sin θ = 0.77379248. The position rotates to Earth-fixed kilometers:

`r_x = −2503.667534 × (−0.63343919) + 4794.241287 × 0.77379248 = 5295.668962`

`r_y = −2503.667534 × 0.77379248 + 4794.241287 × (−0.63343919) = −1099.541215`

`r_z = 4269.926274`

The same rotation applied to the velocity, before the Earth-rate correction, is (−4.820912, −0.287388, 5.879823) km/s. Adding ω × the position:

`v_x = −4.820912 + (7.2921150×10⁻⁵)(−1099.541215) = −4.901092`

`v_y = −0.287388 − (7.2921150×10⁻⁵)(5295.668962) = −0.673554`

`v_z = 5.879823`

`|r| = 6890.963 km`, so the altitude is `6890.963 − 6378.137 = 512.826 km`. The sub-satellite point is latitude 38.289956°, longitude −11.729677°. Lisbon is about 2.6° of longitude to the east of that track.

The point vector, from φ = 38.7223° and λ = −9.1393°:

`p = (4912.967926, −790.386677, 3989.820378)` km

Line of sight from Lisbon up to the satellite, `r − p`:

`(382.701036, −309.154538, 280.105896)` km, length 566.124 km

The three terms of `p · (r − p)` are 1,880,197.9 + 244,351.6 + 1,117,572.2 = 3,242,121.8. Divided by `|p| × 566.124 = 3,610,813.8`:

`cos i = 0.897892`

`i = 26.1176°`

That is inside 20–40°, inside Spot’s requestable 10–45°, and inside the Standard bus reach of 15–40°. Elevation above the Lisbon horizon is `90 − 26.1176 = 63.8824°`. Off-nadir at the satellite is 24.0450°, and `24.0450 + 2.0726 = 26.1176`, the earth-central angle of 2.0726° being 230.72 km of ground range from the track.

`h = r × v = (−3589.082, −52064.898, −8955.872)`

`h · p = −12,213,961.6`

The sign is negative, so Lisbon is to the right of the ground track. The request said `any`. Aether-A1’s antenna looks right, so the side matches. The same second with `"lookSide": "LEFT"` would fail here, and it would fail for this satellite on every other second too.

Northward velocity at the satellite:

`v_north = 7.5036 km/s`

Positive, so the pass is ascending. `"passDirection": "ANY"` accepts it. `"ASCENDING"` would also accept it. `"DESCENDING"` would reject it.

Squint on the seconds around the crossing:

| UTC | Along-track component, km | Squint | Incidence |
| --- | --- | --- | --- |
| 06:06:49 | +15.785 | +3.915° | 26.170° |
| 06:06:50 | +8.698 | +2.160° | 26.133° |
| 06:06:51 | +1.611 | +0.400° | 26.118° |
| 06:06:52 | −5.476 | −1.360° | 26.124° |
| 06:06:53 | −12.563 | −3.118° | 26.153° |

The along-track component changes sign between 06:06:51 and 06:06:52. The smaller absolute squint is 0.400°, which is under 1°, so the start is 06:06:51. The true broadside is a fraction of a second later. The scheduler does not start between seconds.

Spot’s nominal duration is 10 s, so the collection runs until 06:07:01. That instant is inside the window, which ends at 00:00 the next day. The reserved interval is `[2026-10-29T06:06:51Z, 2026-10-29T06:07:01Z)`. Nothing else is reserved, so the satellite is free.

No earlier second on any Spot-capable satellite passed the same list. That is why 06:06:51 is the earliest, not merely a legal time.

The same request with `"lookSide": "LEFT"` has no passing second inside this window. The next left-looking Spot broadside of Lisbon is Aether-B2, which can look either side, at 2026-10-30T10:15:54Z, incidence 23.57°. That is after the window ends, so the left-looking task is not feasible. The point and the mode did not change. The side did.

## Orders

The satellite calendar is the `tasks` table in the constellation database. One row is one task: the satellite, the half-open imaging interval, the collection the constellation chose, and the point and mode the caller asked for. The row stays after the work finishes. A constraint rejects a second busy task that overlaps the same satellite. Cancelled and failed rows do not count. A collection that starts when another ends does not overlap. Two points can be different and still conflict, because the satellite cannot illuminate two scenes at once.

`POST /api/acquisitions` reads that table and writes nothing. It skips cancelled and failed tasks. `POST /api/tasks` finds the earliest free collection and inserts the task in the same transaction. The same `clientRef` returns the existing task. `POST /api/tasks/query` returns the tasks a portal already has, with current status. `POST /api/tasks/{id}/cancel` sets the task to cancelled. It does not delete the row. A task that is already done or failed is refused.

Status is the satellite's work. Constellation is the only writer. Before the imaging start the task is scheduled. From the start until the end it is imaging. After the imaging end it is downlink, then processing, then done. Those two waits are `TASK_DOWNLINK_DELAY` and `TASK_PROCESSING_DELAY`, read when the process starts. The defaults are 15 seconds and 20 seconds. A zero duration skips that phase. The waits do not extend the imaging interval, so the satellite can be offered again while the previous product is still coming down or being processed. A task already marked done stays done. A task whose whole pipeline is already over is stored as done immediately. Cancel sets cancelled, and it is refused once imaging has ended. failed is allowed and nothing sets it yet. A one-second pass updates stored rows that are still scheduled, imaging, downlink, or processing. Reading a task uses the same rule.

The tasking database stores the order and the last status it was told. It does not recompute that status from the clock. Cancel asks the constellation first, then stores the status it returns. A finished task stays done, so that past second is not offered again.
