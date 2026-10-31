import { Component, inject } from '@angular/core';
import { Globe } from './globe';
import { Orbits } from './orbits';
import { SimClock } from './sim-clock';

@Component({
  selector: 'app-satellite-page',
  imports: [Globe],
  templateUrl: './satellite-page.html',
  styleUrl: './satellite-page.css',
})
export class SatellitePage {
  readonly clock = inject(SimClock);
  readonly orbits = inject(Orbits);

  constructor() {
    this.orbits.ensure(Date.now());
  }
}
