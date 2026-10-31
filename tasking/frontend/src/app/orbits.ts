import { HttpClient } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';

export interface Sample {
  t: number;
  ecefX: number;
  ecefY: number;
  ecefZ: number;
}

export interface Track {
  name: string;
  plane: string;
  className: string;
  periodMs: number;
  lookSide: 'right' | 'left' | 'either';
  samples: Sample[];
}

interface CatalogBody {
  planes: { name: string; color: string; periodMin: number }[];
  satellites: { name: string; plane: string; class: string; look: 'right' | 'left' | 'either' }[];
}

interface EphemerisBody {
  satellites: {
    name: string;
    samples: { t: string; ecefX: number; ecefY: number; ecefZ: number }[];
  }[];
}

@Injectable({ providedIn: 'root' })
export class Orbits {
  private readonly http = inject(HttpClient);

  readonly tracks = signal<Track[] | null>(null);
  readonly status = signal<'loading' | 'ready' | 'error'>('loading');
  readonly selected = signal<string | null>(null);
  readonly readout = signal<{ lat: string; lon: string; alt: string } | null>(null);

  private catalog = new Map<string, { plane: string; className: string; lookSide: Track['lookSide'] }>();
  private planes = new Map<string, { color: string; periodMs: number }>();
  private loadedFrom = 0;
  private loadedTo = 0;
  private inflight = false;

  color(plane: string): string {
    return this.planes.get(plane)?.color ?? '#d5dde4';
  }

  ensure(timeMs: number): void {
    const behind = 20 * 60 * 1000;
    const ahead = 110 * 60 * 1000;
    if (this.tracks() && timeMs > this.loadedFrom + behind && timeMs < this.loadedTo - ahead) {
      return;
    }
    if (this.inflight) {
      return;
    }
    this.inflight = true;
    const from = new Date(timeMs - 20 * 60 * 1000);
    const to = new Date(timeMs + 4 * 60 * 60 * 1000);
    const params = `from=${encodeURIComponent(from.toISOString())}&to=${encodeURIComponent(to.toISOString())}&stepSeconds=20`;
    const loadTracks = () => {
      this.http.get<EphemerisBody>(`/api/ephemeris?${params}`).subscribe({
        next: (body) => {
          const tracks = body.satellites.map((sat) => {
            const meta = this.catalog.get(sat.name);
            return {
              name: sat.name,
              plane: meta?.plane ?? '',
              className: meta?.className ?? '',
              periodMs: this.planes.get(meta?.plane ?? '')?.periodMs ?? 95.5 * 60 * 1000,
              lookSide: meta?.lookSide ?? 'either',
              samples: sat.samples.map((sample) => ({
                t: Date.parse(sample.t),
                ecefX: sample.ecefX,
                ecefY: sample.ecefY,
                ecefZ: sample.ecefZ,
              })),
            };
          });
          this.loadedFrom = from.getTime();
          this.loadedTo = to.getTime();
          this.tracks.set(tracks);
          this.status.set('ready');
          this.inflight = false;
        },
        error: () => {
          this.status.set('error');
          this.inflight = false;
        },
      });
    };
    if (this.catalog.size > 0) {
      loadTracks();
      return;
    }
    this.http.get<CatalogBody>('/api/satellites').subscribe({
      next: (body) => {
        for (const plane of body.planes) {
          this.planes.set(plane.name, { color: plane.color, periodMs: plane.periodMin * 60 * 1000 });
        }
        for (const sat of body.satellites) {
          this.catalog.set(sat.name, { plane: sat.plane, className: sat.class, lookSide: sat.look });
        }
        loadTracks();
      },
      error: () => {
        this.status.set('error');
        this.inflight = false;
      },
    });
  }
}

export function sampleAt(samples: Sample[], timeMs: number): Sample | null {
  if (samples.length === 0 || timeMs < samples[0].t || timeMs > samples[samples.length - 1].t) {
    return null;
  }
  let lo = 0;
  let hi = samples.length - 1;
  while (lo + 1 < hi) {
    const mid = (lo + hi) >> 1;
    if (samples[mid].t <= timeMs) {
      lo = mid;
    } else {
      hi = mid;
    }
  }
  const a = samples[lo];
  const b = samples[hi];
  const span = b.t - a.t;
  const u = span === 0 ? 0 : (timeMs - a.t) / span;
  return {
    t: timeMs,
    ecefX: a.ecefX + (b.ecefX - a.ecefX) * u,
    ecefY: a.ecefY + (b.ecefY - a.ecefY) * u,
    ecefZ: a.ecefZ + (b.ecefZ - a.ecefZ) * u,
  };
}

export function velocityAt(samples: Sample[], timeMs: number): { x: number; y: number; z: number } | null {
  if (samples.length < 2 || timeMs < samples[0].t || timeMs > samples[samples.length - 1].t) {
    return null;
  }
  let lo = 0;
  let hi = samples.length - 1;
  while (lo + 1 < hi) {
    const mid = (lo + hi) >> 1;
    if (samples[mid].t <= timeMs) {
      lo = mid;
    } else {
      hi = mid;
    }
  }
  const a = samples[lo];
  const b = samples[hi];
  const dt = (b.t - a.t) / 1000;
  if (dt <= 0) {
    return null;
  }
  return {
    x: (b.ecefX - a.ecefX) / dt,
    y: (b.ecefY - a.ecefY) / dt,
    z: (b.ecefZ - a.ecefZ) / dt,
  };
}

export function ecefToLla(x: number, y: number, z: number): { lat: number; lon: number; altKm: number } {
  return {
    lon: (Math.atan2(y, x) * 180) / Math.PI,
    lat: (Math.atan2(z, Math.hypot(x, y)) * 180) / Math.PI,
    altKm: Math.hypot(x, y, z) - 6378.137,
  };
}
