import { Component, ElementRef, HostListener, OnDestroy, ViewChild, input, output, signal } from '@angular/core';

export interface MenuOption {
  id: string;
  label: string;
}

@Component({
  selector: 'app-menu',
  templateUrl: './menu.html',
  styleUrl: './menu.css',
})
export class Menu implements OnDestroy {
  readonly label = input.required<string>();
  readonly value = input.required<string>();
  readonly options = input.required<MenuOption[]>();
  readonly picked = output<string>();

  readonly open = signal(false);
  readonly flip = signal(false);
  readonly top = signal(0);
  readonly bottom = signal(0);
  readonly left = signal(0);
  readonly width = signal(0);

  @ViewChild('trigger') private trigger?: ElementRef<HTMLButtonElement>;

  private readonly closeOnScroll = (event: Event) => {
    const list = this.trigger?.nativeElement.closest('app-menu')?.querySelector('.list');
    if (list && event.target instanceof Node && list.contains(event.target)) {
      return;
    }
    this.open.set(false);
  };

  constructor() {
    document.addEventListener('scroll', this.closeOnScroll, true);
  }

  ngOnDestroy(): void {
    document.removeEventListener('scroll', this.closeOnScroll, true);
  }

  current(): string {
    return this.options().find((option) => option.id === this.value())?.label ?? '';
  }

  toggle(): void {
    if (this.open()) {
      this.open.set(false);
      return;
    }
    const rect = this.trigger?.nativeElement.getBoundingClientRect();
    if (!rect) {
      return;
    }
    const spaceBelow = window.innerHeight - rect.bottom;
    this.flip.set(spaceBelow < 260);
    this.top.set(rect.bottom + 6);
    this.bottom.set(window.innerHeight - rect.top + 6);
    this.left.set(rect.left);
    this.width.set(rect.width);
    this.open.set(true);
  }

  choose(id: string): void {
    this.picked.emit(id);
    this.open.set(false);
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    if (!this.open()) {
      return;
    }
    const host = this.trigger?.nativeElement.closest('app-menu');
    if (host && !host.contains(event.target as Node)) {
      this.open.set(false);
    }
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.open.set(false);
  }

  @HostListener('window:scroll')
  onScroll(): void {
    this.open.set(false);
  }
}
