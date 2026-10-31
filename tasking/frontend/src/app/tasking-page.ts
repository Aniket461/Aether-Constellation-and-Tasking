import { Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { ImagingMode, ImagingModes } from './imaging-modes';
import { Menu, MenuOption } from './menu';
import { asUtc, Offer, utcStamp } from './order';
import { TaskingGlobe } from './tasking-globe';

const INCIDENCE_OPTIONS: MenuOption[] = [
  { id: 'performant', label: 'Performant band' },
  { id: 'widest', label: 'Widest allowed band' },
];
const LOOK_OPTIONS: MenuOption[] = [
  { id: 'ANY', label: 'Any side' },
  { id: 'LEFT', label: 'Left' },
  { id: 'RIGHT', label: 'Right' },
];
const PASS_OPTIONS: MenuOption[] = [
  { id: 'ANY', label: 'Any direction' },
  { id: 'ASCENDING', label: 'Ascending, northbound' },
  { id: 'DESCENDING', label: 'Descending, southbound' },
];

@Component({
  selector: 'app-tasking-page',
  imports: [TaskingGlobe, Menu],
  templateUrl: './tasking-page.html',
  styleUrl: './tasking-page.css',
})
export class TaskingPage {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  readonly imaging = inject(ImagingModes);

  readonly modeOptions = computed(() => this.imaging.list().map((mode) => ({ id: mode.id, label: mode.label })));
  readonly incidenceOptions = INCIDENCE_OPTIONS;
  readonly lookOptions = LOOK_OPTIONS;
  readonly passOptions = PASS_OPTIONS;
  readonly modeId = signal('');
  readonly look = signal('ANY');
  readonly pass = signal('ANY');
  readonly incidence = signal<'performant' | 'widest'>('performant');
  readonly windowStart = signal(utcStamp(Date.now()));
  readonly windowEnd = signal(utcStamp(Date.now() + 24 * 60 * 60 * 1000));
  readonly poi = signal<{ lat: number; lon: number } | null>(null);
  readonly notice = signal('');
  readonly offer = signal<Offer | null>(null);
  readonly busy = signal(false);

  constructor() {
    effect(() => {
      const list = this.imaging.list();
      if (list.length > 0 && !list.some((mode) => mode.id === this.modeId())) {
        this.modeId.set(list[0].id);
      }
    });
    effect(() => {
      this.modeId();
      this.look();
      this.pass();
      this.incidence();
      this.windowStart();
      this.windowEnd();
      this.poi();
      untracked(() => this.offer.set(null));
    });
  }

  readonly mode = computed((): ImagingMode => {
    const list = this.imaging.list();
    return (
      list.find((item) => item.id === this.modeId()) ??
      list[0] ?? {
        id: '',
        label: '',
        seconds: 10,
        rangeKm: 5,
        azimuthKm: 5,
        performMin: 20,
        performMax: 40,
        requestMin: 10,
        requestMax: 45,
        classes: [],
      }
    );
  });
  readonly place = computed(() => {
    const poi = this.poi();
    if (!poi) {
      return 'Click the globe';
    }
    return `${poi.lat.toFixed(4)}°, ${poi.lon.toFixed(4)}°`;
  });
  readonly incidenceText = computed(() => {
    const mode = this.mode();
    const [lo, hi] =
      this.incidence() === 'performant'
        ? [mode.performMin, mode.performMax]
        : [mode.requestMin, mode.requestMax];
    return `${lo}–${hi}°`;
  });

  pick(poi: { lat: number; lon: number }): void {
    this.poi.set(poi);
    this.notice.set('');
  }

  check(): void {
    const poi = this.poi();
    if (!poi) {
      this.notice.set('Click the globe to choose a place first.');
      return;
    }
    this.busy.set(true);
    this.notice.set('');
    this.http.post<Offer>('/api/feasibility', this.body(poi)).subscribe({
      next: (result) => {
        this.busy.set(false);
        if (!result.feasible || !result.imagingTime) {
          this.notice.set('No satellite can take this picture in that window.');
          return;
        }
        this.offer.set(result);
      },
      error: () => {
        this.busy.set(false);
        this.notice.set('The tasking service did not answer.');
      },
    });
  }

  placeOrder(): void {
    const poi = this.poi();
    if (!poi || !this.offer()) {
      return;
    }
    this.busy.set(true);
    this.http.post<{ id: string }>('/api/orders', this.body(poi)).subscribe({
      next: (order) => {
        this.busy.set(false);
        void this.router.navigate(['/orders', order.id]);
      },
      error: () => {
        this.busy.set(false);
        this.offer.set(null);
        this.notice.set('That slot was just taken. Check again.');
      },
    });
  }

  reset(): void {
    const first = this.imaging.list()[0];
    this.modeId.set(first ? first.id : '');
    this.look.set('ANY');
    this.pass.set('ANY');
    this.incidence.set('performant');
    this.windowStart.set(utcStamp(Date.now()));
    this.windowEnd.set(utcStamp(Date.now() + 24 * 60 * 60 * 1000));
    this.poi.set(null);
    this.notice.set('');
  }

  private body(poi: { lat: number; lon: number }) {
    return {
      pointOfInterest: poi,
      acquisitionWindow: { start: asUtc(this.windowStart()), end: asUtc(this.windowEnd()) },
      imagingMode: this.modeId(),
      lookSide: this.look(),
      passDirection: this.pass(),
      incidence: this.incidence(),
    };
  }

  asText(event: Event): string {
    return (event.target as HTMLInputElement).value;
  }

  setIncidence(id: string): void {
    this.incidence.set(id === 'widest' ? 'widest' : 'performant');
  }
}
