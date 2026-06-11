// Date helpers: API speaks RFC3339, <input type="datetime-local"> speaks
// local "YYYY-MM-DDTHH:mm".

export function toLocalInput(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function fromLocalInput(v: string): string | undefined {
  if (!v) return undefined;
  return new Date(v).toISOString();
}

export interface DueLabel {
  text: string;
  tone: 'overdue' | 'today' | 'future';
}

export function dueLabel(iso?: string): DueLabel | null {
  if (!iso) return null;
  const due = new Date(iso);
  const now = new Date();
  const endOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59);
  if (due.getTime() < now.getTime() - 60_000 && due < endOfToday) {
    const days = Math.floor((now.getTime() - due.getTime()) / 86_400_000);
    return { text: days >= 1 ? `${days}d overdue` : 'overdue', tone: 'overdue' };
  }
  if (due <= endOfToday) return { text: 'today', tone: 'today' };
  const days = Math.ceil((due.getTime() - endOfToday.getTime()) / 86_400_000);
  if (days <= 14) return { text: `${days}d`, tone: 'future' };
  return {
    text: due.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    tone: 'future',
  };
}

// isDeferred: hidden by the GTD tickler until this time passes.
export function isDeferred(iso?: string): boolean {
  return !!iso && new Date(iso) > new Date();
}
