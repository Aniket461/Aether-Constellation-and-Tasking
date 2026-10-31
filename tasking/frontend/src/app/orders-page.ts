import { Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { RouterLink } from '@angular/router';
import { ImagingModes } from './imaging-modes';
import { Order, OrderState, isOpen, statusLabel } from './order';
import { SimClock } from './sim-clock';

@Component({
  selector: 'app-orders-page',
  imports: [RouterLink],
  templateUrl: './orders-page.html',
  styleUrl: './orders-page.css',
})
export class OrdersPage {
  private readonly http = inject(HttpClient);
  readonly clock = inject(SimClock);
  private readonly imaging = inject(ImagingModes);
  readonly orders = signal<Order[] | null>(null);
  readonly failed = signal(false);

  readonly rows = computed(() => {
    const orders = this.orders();
    if (!orders) {
      return [];
    }
    return orders.map((order) => ({
      order,
      state: order.status,
    }));
  });

  constructor() {
    this.load(true);
    const timer = window.setInterval(() => {
      const orders = this.orders();
      if (orders?.some((order) => isOpen(order.status))) {
        this.load(false);
      }
    }, 1000);
    inject(DestroyRef).onDestroy(() => window.clearInterval(timer));
  }

  private load(initial: boolean): void {
    this.http.get<Order[]>('/api/orders').subscribe({
      next: (orders) => this.orders.set(orders),
      error: () => {
        if (initial) {
          this.failed.set(true);
        }
      },
    });
  }

  label(state: OrderState): string {
    return statusLabel(state);
  }

  modeLabel(id: string): string {
    return this.imaging.label(id);
  }
}
