import { Injectable, NgZone, signal } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class SimClock {
  readonly label = signal(formatUtc(Date.now()));

  currentMs = Date.now();

  private lastPaint = 0;

  constructor(zone: NgZone) {
    zone.runOutsideAngular(() => {
      const tick = (now: number) => {
        this.currentMs = Date.now();
        if (now - this.lastPaint > 200) {
          this.lastPaint = now;
          const text = formatUtc(this.currentMs);
          zone.run(() => this.label.set(text));
        }
        requestAnimationFrame(tick);
      };
      requestAnimationFrame(tick);
    });
  }
}

function formatUtc(ms: number): string {
  const date = new Date(ms);
  const part = (value: number) => String(value).padStart(2, '0');
  return `${date.getUTCFullYear()}-${part(date.getUTCMonth() + 1)}-${part(date.getUTCDate())} ${part(date.getUTCHours())}:${part(date.getUTCMinutes())}:${part(date.getUTCSeconds())} UTC`;
}
