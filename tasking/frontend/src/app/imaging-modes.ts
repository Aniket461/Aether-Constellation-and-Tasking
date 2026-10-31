import { HttpClient } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';

export interface ImagingMode {
  id: string;
  label: string;
  seconds: number;
  rangeKm: number;
  azimuthKm: number;
  performMin: number;
  performMax: number;
  requestMin: number;
  requestMax: number;
  classes: string[];
}

@Injectable({ providedIn: 'root' })
export class ImagingModes {
  private readonly http = inject(HttpClient);
  readonly list = signal<ImagingMode[]>([]);
  readonly failed = signal(false);

  constructor() {
    this.http.get<ImagingMode[]>('/api/modes').subscribe({
      next: (modes) => this.list.set(modes),
      error: () => this.failed.set(true),
    });
  }

  label(id: string): string {
    return this.list().find((mode) => mode.id === id)?.label ?? id;
  }
}
