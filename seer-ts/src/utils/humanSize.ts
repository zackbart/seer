const units = ["B", "KB", "MB", "GB", "TB"];

export function humanSize(n: number): string {
  let v = n;
  let idx = 0;
  while (v >= 1024 && idx < units.length - 1) {
    v /= 1024;
    idx++;
  }
  if (idx === 0) return `${n} ${units[idx]}`;
  return `${v.toFixed(1)} ${units[idx]}`;
}

export function previewPageSize(h: number): number {
  return Math.max(3, Math.floor(h / 3));
}
