import dayjs from 'dayjs';

// Shared presentation helpers for the mail app: deterministic sender avatars,
// relative/exact timestamps, and attachment chip formatting.

const AVATAR_COLORS = ['neonPink', 'electricBlue', 'synthPurple', 'neonGreen', 'hotYellow'];

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

// senderColor maps a stable key (email/name) to one of the theme palettes.
export function senderColor(key: string): string {
  let h = 0;
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0;
  return AVATAR_COLORS[h % AVATAR_COLORS.length];
}

export function relativeTime(iso: string): string {
  return dayjs(iso).fromNow(); // dayjs.extend(relativeTime) is called in main.tsx
}

export function exactTime(iso: string): string {
  return dayjs(iso).format('ddd, MMM D, YYYY · h:mm A');
}

export function formatBytes(n: number): string {
  if (n <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.min(units.length - 1, Math.floor(Math.log(n) / Math.log(1024)));
  const v = n / Math.pow(1024, i);
  return `${v >= 10 || i === 0 ? Math.round(v) : v.toFixed(1)} ${units[i]}`;
}

// attachmentIcon returns an emoji glyph for an attachment's content type.
export function attachmentIcon(contentType: string): string {
  const ct = contentType.toLowerCase();
  if (ct.includes('pdf')) return '📄';
  if (ct.startsWith('image/')) return '🖼️';
  if (ct.includes('spreadsheet') || ct.includes('excel') || ct.includes('csv')) return '📊';
  if (ct.includes('presentation') || ct.includes('powerpoint')) return '📈';
  if (ct.includes('zip') || ct.includes('compressed')) return '🗜️';
  if (ct.includes('word') || ct.includes('document')) return '📝';
  return '📎';
}
