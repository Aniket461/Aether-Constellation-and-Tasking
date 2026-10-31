export interface TimeSpan {
  start: string;
  end: string;
}

export interface Offer {
  feasible: boolean;
  satellite?: string;
  plane?: string;
  class?: string;
  imagingTime?: TimeSpan;
  durationSeconds?: number;
  incidenceAngle?: number;
  lookSide?: string;
  passDirection?: string;
  squintDeg?: number;
}

export interface Order {
  id: string;
  taskId: string;
  status: OrderState;
  createdAt: string;
  cancelledAt: string | null;
  satellite: string;
  plane: string;
  class: string;
  pointOfInterest: { lat: number; lon: number };
  imagingMode: string;
  lookSide: string;
  passDirection: string;
  incidence: string;
  acquisitionWindow: TimeSpan;
  imagingTime: TimeSpan;
  durationSeconds: number;
  incidenceAngle: number;
  computedLookSide: string;
  computedPass: string;
  squintDeg: number;
}

export type OrderState = 'scheduled' | 'imaging' | 'downlink' | 'processing' | 'done' | 'cancelled' | 'failed';

export function statusLabel(state: OrderState | ''): string {
  switch (state) {
    case 'scheduled':
      return 'Scheduled';
    case 'imaging':
      return 'Imaging';
    case 'downlink':
      return 'Downlink';
    case 'processing':
      return 'Processing';
    case 'done':
      return 'Done';
    case 'cancelled':
      return 'Cancelled';
    case 'failed':
      return 'Failed';
    default:
      return '';
  }
}

export function isOpen(state: OrderState | ''): boolean {
  return state === 'scheduled' || state === 'imaging' || state === 'downlink' || state === 'processing';
}

export function utcStamp(ms: number): string {
  const date = new Date(ms);
  const part = (value: number) => String(value).padStart(2, '0');
  return `${date.getUTCFullYear()}-${part(date.getUTCMonth() + 1)}-${part(date.getUTCDate())}T${part(date.getUTCHours())}:${part(date.getUTCMinutes())}`;
}

export function asUtc(value: string): string {
  const withSeconds = value.length === 16 ? `${value}:00` : value;
  return withSeconds.endsWith('Z') ? withSeconds : `${withSeconds}Z`;
}
