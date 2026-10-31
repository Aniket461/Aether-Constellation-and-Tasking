import {
  AfterViewInit,
  Component,
  ElementRef,
  NgZone,
  OnDestroy,
  ViewChild,
  inject,
} from '@angular/core';
import {
  ArcType,
  CallbackProperty,
  Cartesian2,
  Cartesian3,
  Color,
  ConstantPositionProperty,
  ConstantProperty,
  HorizontalOrigin,
  ImageryLayer,
  JulianDate,
  LabelStyle,
  PolygonHierarchy,
  UrlTemplateImageryProvider,
  VerticalOrigin,
  Viewer,
  type Entity,
} from 'cesium';
import { Orbits, Track, ecefToLla, sampleAt, velocityAt } from './orbits';
import { SimClock } from './sim-clock';

interface Drawn {
  track: Track;
  point: Entity;
  orbit: Entity;
  beam: Entity;
  sheetAlong: Entity;
  sheetAcross: Entity;
  footprint: Entity;
  position: Cartesian3;
  aim: Cartesian3;
  beamFrom: Cartesian3;
  beamTo: Cartesian3;
  alongTri: Cartesian3[];
  acrossTri: Cartesian3[];
}

// Sidereal Earth rate. Used to draw one inertial revolution in the current Earth frame.
const EARTH_RAD_PER_SEC = 7.292115e-5;

@Component({
  selector: 'app-globe',
  template: '<div #host class="host"></div>',
  styles: `
    :host {
      display: block;
      height: 100%;
    }
    .host {
      height: 100%;
    }
  `,
})
export class Globe implements AfterViewInit, OnDestroy {
  private readonly zone = inject(NgZone);
  private readonly clock = inject(SimClock);
  private readonly orbits = inject(Orbits);

  @ViewChild('host', { static: true }) private host!: ElementRef<HTMLDivElement>;

  private viewer: Viewer | null = null;
  private drawn: Drawn[] = [];
  private drawnFrom: Track[] | null = null;
  private frame = 0;
  private lastTrail = 0;
  private lastReadout = 0;

  ngAfterViewInit(): void {
    const imagery = new UrlTemplateImageryProvider({
      url: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
      credit: 'Esri, Maxar, Earthstar Geographics',
    });
    this.viewer = new Viewer(this.host.nativeElement, {
      animation: false,
      timeline: false,
      baseLayerPicker: false,
      geocoder: false,
      homeButton: false,
      sceneModePicker: false,
      navigationHelpButton: false,
      fullscreenButton: false,
      infoBox: false,
      baseLayer: new ImageryLayer(imagery),
    });
    const viewer = this.viewer;
    viewer.clock.shouldAnimate = false;
    viewer.scene.globe.enableLighting = true;
    viewer.camera.setView({ destination: Cartesian3.fromDegrees(10, 18, 20_000_000) });
    viewer.selectedEntityChanged.addEventListener((entity: Entity | undefined) => {
      this.zone.run(() => this.orbits.selected.set(entity?.name ?? null));
    });

    this.zone.runOutsideAngular(() => {
      const tick = (now: number) => {
        this.frame = requestAnimationFrame(tick);
        this.paint(now);
      };
      this.frame = requestAnimationFrame(tick);
    });
  }

  ngOnDestroy(): void {
    cancelAnimationFrame(this.frame);
    this.viewer?.destroy();
    this.viewer = null;
  }

  private paint(now: number): void {
    const viewer = this.viewer;
    if (!viewer) {
      return;
    }
    const timeMs = this.clock.currentMs;
    this.orbits.ensure(timeMs);
    const tracks = this.orbits.tracks();
    if (!tracks) {
      return;
    }
    this.sync(viewer, tracks);
    viewer.clock.currentTime = JulianDate.fromDate(new Date(timeMs));

    const selected = this.orbits.selected();
    if ((viewer.selectedEntity?.name ?? null) !== selected) {
      viewer.selectedEntity = this.drawn.find((item) => item.track.name === selected)?.point;
    }

    const refreshOrbit = now - this.lastTrail > 1000;
    if (refreshOrbit) {
      this.lastTrail = now;
    }
    let readout: { lat: number; lon: number; altKm: number } | null = null;

    for (const item of this.drawn) {
      const sample = sampleAt(item.track.samples, timeMs);
      if (!sample) {
        continue;
      }
      Cartesian3.fromElements(sample.ecefX * 1000, sample.ecefY * 1000, sample.ecefZ * 1000, item.position);
      (item.point.position as ConstantPositionProperty).setValue(item.position);
      const on = item.track.name === selected;
      item.point.billboard!.scale = new ConstantProperty(on ? 1.45 : 1);
      item.point.label!.show = new ConstantProperty(on);
      if (on) {
        readout = ecefToLla(sample.ecefX, sample.ecefY, sample.ecefZ);
      }
      if (refreshOrbit && item.orbit.polyline) {
        item.orbit.polyline.positions = new ConstantProperty(orbitRing(item.track, timeMs));
      }
      const velocity = velocityAt(item.track.samples, timeMs);
      if (velocity) {
        paintBeam(item, velocity);
      }
    }

    if (now - this.lastReadout > 200) {
      this.lastReadout = now;
      const text = readout
        ? {
            lat: `${readout.lat.toFixed(2)}°`,
            lon: `${readout.lon.toFixed(2)}°`,
            alt: `${readout.altKm.toFixed(0)} km`,
          }
        : null;
      this.zone.run(() => this.orbits.readout.set(text));
    }
  }

  private sync(viewer: Viewer, tracks: Track[]): void {
    if (this.drawnFrom === tracks) {
      return;
    }
    viewer.entities.removeAll();
    this.drawn = tracks.map((track) => {
      const color = Color.fromCssColorString(this.orbits.color(track.plane));
      const position = new Cartesian3();
      const point = viewer.entities.add({
        name: track.name,
        position: new ConstantPositionProperty(position),
        billboard: {
          image: 'satellite.svg',
          width: 36,
          height: 36,
          color,
          horizontalOrigin: HorizontalOrigin.CENTER,
          verticalOrigin: VerticalOrigin.CENTER,
          disableDepthTestDistance: Number.POSITIVE_INFINITY,
        },
        label: {
          text: track.name,
          font: '13px sans-serif',
          fillColor: Color.WHITE,
          outlineColor: Color.BLACK,
          outlineWidth: 2,
          style: LabelStyle.FILL_AND_OUTLINE,
          pixelOffset: new Cartesian2(24, 0),
          disableDepthTestDistance: Number.POSITIVE_INFINITY,
          show: false,
        },
      });
      const orbit = viewer.entities.add({
        polyline: {
          positions: [],
          width: 1.5,
          material: color.withAlpha(0.9),
          arcType: ArcType.NONE,
        },
      });
      const aim = new Cartesian3();
      const beamFrom = new Cartesian3();
      const beamTo = new Cartesian3();
      const alongTri = [new Cartesian3(), new Cartesian3(), new Cartesian3()];
      const acrossTri = [new Cartesian3(), new Cartesian3(), new Cartesian3()];
      const red = Color.fromCssColorString('#ff3b30');
      const beam = viewer.entities.add({
        name: track.name,
        polyline: {
          positions: new CallbackProperty(() => [beamFrom, beamTo], false),
          width: 5,
          material: red.withAlpha(0.95),
          arcType: ArcType.NONE,
        },
      });
      const sheetAlong = viewer.entities.add({
        name: track.name,
        polygon: {
          hierarchy: new CallbackProperty(() => new PolygonHierarchy(alongTri), false),
          perPositionHeight: true,
          material: red.withAlpha(0.35),
          outline: false,
          arcType: ArcType.NONE,
        },
      });
      const sheetAcross = viewer.entities.add({
        name: track.name,
        polygon: {
          hierarchy: new CallbackProperty(() => new PolygonHierarchy(acrossTri), false),
          perPositionHeight: true,
          material: red.withAlpha(0.22),
          outline: false,
          arcType: ArcType.NONE,
        },
      });
      const footprint = viewer.entities.add({
        name: track.name,
        position: new ConstantPositionProperty(aim),
        point: {
          pixelSize: 8,
          color: red,
          outlineColor: Color.WHITE,
          outlineWidth: 1,
        },
        ellipse: {
          semiMajorAxis: 80_000,
          semiMinorAxis: 50_000,
          material: red.withAlpha(0.45),
          height: 0,
          rotation: new ConstantProperty(0),
        },
      });
      return {
        track,
        point,
        orbit,
        beam,
        sheetAlong,
        sheetAcross,
        footprint,
        position,
        aim,
        beamFrom,
        beamTo,
        alongTri,
        acrossTri,
      };
    });
    this.drawnFrom = tracks;
  }
}

// One revolution, turned into the Earth frame at timeMs so the path is a closed ring.
function orbitRing(track: Track, timeMs: number): Cartesian3[] {
  const end = timeMs + track.periodMs;
  const positions: Cartesian3[] = [];
  for (const sample of track.samples) {
    if (sample.t < timeMs || sample.t > end) {
      continue;
    }
    const angle = EARTH_RAD_PER_SEC * ((sample.t - timeMs) / 1000);
    const cos = Math.cos(angle);
    const sin = Math.sin(angle);
    positions.push(
      Cartesian3.fromElements(
        (sample.ecefX * cos - sample.ecefY * sin) * 1000,
        (sample.ecefX * sin + sample.ecefY * cos) * 1000,
        sample.ecefZ * 1000,
      ),
    );
  }
  if (positions.length > 2) {
    positions.push(Cartesian3.clone(positions[0]));
  }
  return positions;
}

const EARTH_RADIUS_KM = 6378.137;
const BEAM_INCIDENCE = (30 * Math.PI) / 180;
const FOOTPRINT_ALONG_KM = 80;
const FOOTPRINT_ACROSS_KM = 50;

function paintBeam(item: Drawn, velocity: { x: number; y: number; z: number }): void {
  const beam = sideBeam(
    item.position,
    velocity,
    item.track.lookSide === 'left' ? -1 : 1,
  );
  if (!beam) {
    return;
  }
  Cartesian3.clone(beam.aim, item.aim);
  Cartesian3.clone(item.position, item.beamFrom);
  Cartesian3.clone(beam.aim, item.beamTo);
  writeTri(item.alongTri, item.position, beam.ahead, beam.behind);
  writeTri(item.acrossTri, item.position, beam.left, beam.right);
  (item.footprint.position as ConstantPositionProperty).setValue(item.aim);
  item.footprint.ellipse!.rotation = new ConstantProperty(beam.rotation);
}

// Broadside beam at 30° incidence. sideSign +1 is right of the velocity, −1 is left.
function sideBeam(
  satMeters: Cartesian3,
  velocity: { x: number; y: number; z: number },
  sideSign: number,
): { aim: Cartesian3; ahead: Cartesian3; behind: Cartesian3; left: Cartesian3; right: Cartesian3; rotation: number } | null {
  const rx = satMeters.x / 1000;
  const ry = satMeters.y / 1000;
  const rz = satMeters.z / 1000;
  const rMag = Math.hypot(rx, ry, rz);
  if (rMag === 0) {
    return null;
  }
  const urx = rx / rMag;
  const ury = ry / rMag;
  const urz = rz / rMag;
  const climb = velocity.x * urx + velocity.y * ury + velocity.z * urz;
  let ax = velocity.x - climb * urx;
  let ay = velocity.y - climb * ury;
  let az = velocity.z - climb * urz;
  const aMag = Math.hypot(ax, ay, az);
  if (aMag < 1e-6) {
    return null;
  }
  ax /= aMag;
  ay /= aMag;
  az /= aMag;
  const sx = sideSign * (ay * urz - az * ury);
  const sy = sideSign * (az * urx - ax * urz);
  const sz = sideSign * (ax * ury - ay * urx);
  const offNadir = Math.asin(Math.min(1, (EARTH_RADIUS_KM / rMag) * Math.sin(BEAM_INCIDENCE)));
  const central = BEAM_INCIDENCE - offNadir;
  const aim = shift(urx, ury, urz, sx, sy, sz, central);
  const along = FOOTPRINT_ALONG_KM / EARTH_RADIUS_KM;
  const across = FOOTPRINT_ACROSS_KM / EARTH_RADIUS_KM;
  const ux = aim.x / EARTH_RADIUS_KM;
  const uy = aim.y / EARTH_RADIUS_KM;
  const uz = aim.z / EARTH_RADIUS_KM;
  const aheadAxis = tangent(ax, ay, az, ux, uy, uz);
  const acrossAxis = tangent(sx, sy, sz, ux, uy, uz);
  if (!aheadAxis || !acrossAxis) {
    return null;
  }
  return {
    aim: kmToMeters(aim),
    ahead: kmToMeters(shift(ux, uy, uz, aheadAxis[0], aheadAxis[1], aheadAxis[2], along)),
    behind: kmToMeters(shift(ux, uy, uz, -aheadAxis[0], -aheadAxis[1], -aheadAxis[2], along)),
    right: kmToMeters(shift(ux, uy, uz, acrossAxis[0], acrossAxis[1], acrossAxis[2], across)),
    left: kmToMeters(shift(ux, uy, uz, -acrossAxis[0], -acrossAxis[1], -acrossAxis[2], across)),
    rotation: trackRotation(ux, uy, uz, aheadAxis[0], aheadAxis[1], aheadAxis[2]),
  };
}

function shift(
  ux: number,
  uy: number,
  uz: number,
  dx: number,
  dy: number,
  dz: number,
  angle: number,
): { x: number; y: number; z: number } {
  const cos = Math.cos(angle);
  const sin = Math.sin(angle);
  return {
    x: EARTH_RADIUS_KM * (cos * ux + sin * dx),
    y: EARTH_RADIUS_KM * (cos * uy + sin * dy),
    z: EARTH_RADIUS_KM * (cos * uz + sin * dz),
  };
}

function tangent(
  ax: number,
  ay: number,
  az: number,
  nx: number,
  ny: number,
  nz: number,
): [number, number, number] | null {
  const dot = ax * nx + ay * ny + az * nz;
  const x = ax - dot * nx;
  const y = ay - dot * ny;
  const z = az - dot * nz;
  const mag = Math.hypot(x, y, z);
  if (mag < 1e-6) {
    return null;
  }
  return [x / mag, y / mag, z / mag];
}

function kmToMeters(p: { x: number; y: number; z: number }): Cartesian3 {
  return Cartesian3.fromElements(p.x * 1000, p.y * 1000, p.z * 1000);
}

function trackRotation(ux: number, uy: number, uz: number, ax: number, ay: number, az: number): number {
  const eastMag = Math.hypot(-uy, ux);
  if (eastMag < 1e-6) {
    return 0;
  }
  const ex = -uy / eastMag;
  const ey = ux / eastMag;
  const nx = uy * 0 - uz * ey;
  const ny = uz * ex - ux * 0;
  const nz = ux * ey - uy * ex;
  return Math.atan2(ax * ex + ay * ey, ax * nx + ay * ny + az * nz);
}

function writeTri(dest: Cartesian3[], sat: Cartesian3, a: Cartesian3, b: Cartesian3): void {
  const ordered = faceOut(sat, a, b);
  Cartesian3.clone(ordered[0], dest[0]);
  Cartesian3.clone(ordered[1], dest[1]);
  Cartesian3.clone(ordered[2], dest[2]);
}

function faceOut(sat: Cartesian3, a: Cartesian3, b: Cartesian3): Cartesian3[] {
  const ax = a.x - sat.x;
  const ay = a.y - sat.y;
  const az = a.z - sat.z;
  const bx = b.x - sat.x;
  const by = b.y - sat.y;
  const bz = b.z - sat.z;
  const nx = ay * bz - az * by;
  const ny = az * bx - ax * bz;
  const nz = ax * by - ay * bx;
  if (nx * sat.x + ny * sat.y + nz * sat.z < 0) {
    return [sat, b, a];
  }
  return [sat, a, b];
}
