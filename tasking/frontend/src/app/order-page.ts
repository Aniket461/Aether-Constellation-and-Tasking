import { Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ImagingModes } from './imaging-modes';
import { Order, isOpen } from './order';
import { SimClock } from './sim-clock';

@Component({
  selector: 'app-order-page',
  imports: [RouterLink],
  templateUrl: './order-page.html',
  styleUrl: './order-page.css',
})
export class OrderPage {
  private readonly http = inject(HttpClient);
  private readonly route = inject(ActivatedRoute);
  readonly clock = inject(SimClock);
  private readonly imaging = inject(ImagingModes);
  readonly order = signal<Order | null>(null);
  readonly failed = signal('');

  readonly state = computed(() => this.order()?.status ?? '');

  readonly steps = [
    { id: 'scheduled', label: 'Scheduled' },
    { id: 'imaging', label: 'Imaging' },
    { id: 'downlink', label: 'Downlink' },
    { id: 'processing', label: 'Processing' },
    { id: 'done', label: 'Done' },
  ] as const;

  readonly stepIndex = computed(() => {
    const order = this.order();
    const state = this.state();
    if (!order || !state) {
      return 0;
    }
    if ((state === 'cancelled' || state === 'failed') && order.cancelledAt) {
      return Date.parse(order.cancelledAt) < Date.parse(order.imagingTime.start) ? 0 : 1;
    }
    if (state === 'imaging' || state === 'failed') {
      return 1;
    }
    if (state === 'downlink') {
      return 2;
    }
    if (state === 'processing') {
      return 3;
    }
    if (state === 'done') {
      return 4;
    }
    return 0;
  });

  constructor() {
    const destroy = inject(DestroyRef);
    this.route.paramMap.pipe(takeUntilDestroyed()).subscribe((params) => {
      this.load(params.get('orderId') ?? '', true);
    });
    const timer = window.setInterval(() => {
      const order = this.order();
      if (order && isOpen(order.status)) {
        this.load(order.id, false);
      }
    }, 1000);
    destroy.onDestroy(() => window.clearInterval(timer));
  }

  modeLabel(id: string): string {
    return this.imaging.label(id);
  }

  cancel(): void {
    const order = this.order();
    if (!order) {
      return;
    }
    this.http.post<Order>(`/api/orders/${order.id}/cancel`, {}).subscribe({
      next: (updated) => this.order.set(updated),
      error: () => this.failed.set('This order could not be cancelled.'),
    });
  }

  private load(id: string, initial: boolean): void {
    if (initial) {
      this.order.set(null);
      this.failed.set('');
    }
    this.http.get<Order>(`/api/orders/${id}`).subscribe({
      next: (order) => this.order.set(order),
      error: () => {
        if (initial) {
          this.failed.set('That order was not found.');
        }
      },
    });
  }
}
