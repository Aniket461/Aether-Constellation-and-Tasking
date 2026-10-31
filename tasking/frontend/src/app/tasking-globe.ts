import {
  AfterViewInit,
  Component,
  ElementRef,
  NgZone,
  OnDestroy,
  ViewChild,
  effect,
  inject,
  input,
  output,
} from '@angular/core';
import {
  ArcType,
  CallbackProperty,
  Cartesian2,
  Cartesian3,
  Cartographic,
  Color,
  ConstantPositionProperty,
  Ellipsoid,
  ImageryLayer,
  Math as CesiumMath,
  Matrix4,
  PolygonHierarchy,
  ScreenSpaceEventHandler,
  ScreenSpaceEventType,
  Transforms,
  UrlTemplateImageryProvider,
  Viewer,
  type Entity,
} from 'cesium';

@Component({
  selector: 'app-tasking-globe',
  template: '<div #host class="host"></div>',
  styles: `
    :host,
    .host {
      display: block;
      height: 100%;
    }
  `,
})
export class TaskingGlobe implements AfterViewInit, OnDestroy {
  readonly poi = input<{ lat: number; lon: number } | null>(null);
  readonly rangeKm = input(5);
  readonly azimuthKm = input(5);
  readonly picked = output<{ lat: number; lon: number }>();

  private readonly zone = inject(NgZone);
  @ViewChild('host', { static: true }) private host!: ElementRef<HTMLDivElement>;

  private viewer: Viewer | null = null;
  private handler: ScreenSpaceEventHandler | null = null;
  private marker: Entity | null = null;
  private readonly corners = [new Cartesian3(), new Cartesian3(), new Cartesian3(), new Cartesian3()];
  private flownTo: { lat: number; lon: number } | null = null;

  constructor() {
    effect(() => {
      const poi = this.poi();
      const rangeKm = this.rangeKm();
      const azimuthKm = this.azimuthKm();
      const viewer = this.viewer;
      if (!viewer) {
        return;
      }
      this.show(viewer, poi, rangeKm, azimuthKm);
    });
  }

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
    viewer.scene.globe.enableLighting = false;
    viewer.scene.canvas.style.cursor = 'crosshair';
    viewer.camera.setView({ destination: Cartesian3.fromDegrees(10, 20, 18_000_000) });
    this.show(viewer, this.poi(), this.rangeKm(), this.azimuthKm());

    this.handler = new ScreenSpaceEventHandler(viewer.scene.canvas);
    this.handler.setInputAction((event: { position: Cartesian2 }) => {
      const cartesian = viewer.camera.pickEllipsoid(event.position, Ellipsoid.WGS84);
      if (!cartesian) {
        return;
      }
      const carto = Cartographic.fromCartesian(cartesian);
      const lat = CesiumMath.toDegrees(carto.latitude);
      const lon = CesiumMath.toDegrees(carto.longitude);
      this.zone.run(() => this.picked.emit({ lat, lon }));
    }, ScreenSpaceEventType.LEFT_CLICK);
  }

  ngOnDestroy(): void {
    this.handler?.destroy();
    this.viewer?.destroy();
    this.viewer = null;
    this.marker = null;
  }

  private show(
    viewer: Viewer,
    poi: { lat: number; lon: number } | null,
    rangeKm: number,
    azimuthKm: number,
  ): void {
    if (!poi) {
      if (this.marker) {
        viewer.entities.remove(this.marker);
        this.marker = null;
      }
      this.flownTo = null;
      return;
    }
    const position = Cartesian3.fromDegrees(poi.lon, poi.lat);
    writeScene(poi.lat, poi.lon, rangeKm, azimuthKm, this.corners);
    if (!this.marker) {
      const corners = this.corners;
      const red = Color.fromCssColorString('#ff3b30');
      this.marker = viewer.entities.add({
        position: new ConstantPositionProperty(position),
        point: {
          pixelSize: 6,
          color: Color.WHITE,
          outlineColor: red,
          outlineWidth: 2,
          disableDepthTestDistance: Number.POSITIVE_INFINITY,
        },
        polygon: {
          hierarchy: new CallbackProperty(() => new PolygonHierarchy(corners), false),
          material: red.withAlpha(0.35),
          perPositionHeight: true,
          arcType: ArcType.NONE,
        },
        polyline: {
          positions: new CallbackProperty(() => [corners[0], corners[1], corners[2], corners[3], corners[0]], false),
          width: 2,
          material: red.withAlpha(0.95),
          arcType: ArcType.NONE,
        },
      });
    }
    (this.marker.position as ConstantPositionProperty).setValue(position);
    if (this.flownTo && this.flownTo.lat === poi.lat && this.flownTo.lon === poi.lon) {
      return;
    }
    this.flownTo = { lat: poi.lat, lon: poi.lon };
    viewer.camera.flyTo({
      destination: Cartesian3.fromDegrees(poi.lon, poi.lat, 280_000),
      duration: 1.1,
    });
  }
}

function writeScene(
  lat: number,
  lon: number,
  rangeKm: number,
  azimuthKm: number,
  corners: Cartesian3[],
): void {
  const frame = Transforms.eastNorthUpToFixedFrame(Cartesian3.fromDegrees(lon, lat));
  const halfEast = rangeKm * 500;
  const halfNorth = azimuthKm * 500;
  const offsets: [number, number][] = [
    [halfEast, halfNorth],
    [-halfEast, halfNorth],
    [-halfEast, -halfNorth],
    [halfEast, -halfNorth],
  ];
  for (let i = 0; i < offsets.length; i++) {
    const [east, north] = offsets[i];
    Matrix4.multiplyByPoint(frame, new Cartesian3(east, north, 0), corners[i]);
    Ellipsoid.WGS84.scaleToGeodeticSurface(corners[i], corners[i]);
  }
}
