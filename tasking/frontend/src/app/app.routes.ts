import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: 'satellite', loadComponent: () => import('./satellite-page').then((m) => m.SatellitePage) },
  { path: 'tasking', loadComponent: () => import('./tasking-page').then((m) => m.TaskingPage) },
  { path: 'orders', loadComponent: () => import('./orders-page').then((m) => m.OrdersPage) },
  { path: 'orders/:orderId', loadComponent: () => import('./order-page').then((m) => m.OrderPage) },
  { path: '', pathMatch: 'full', redirectTo: 'tasking' },
];
