import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBackendDate(dateStr?: string | null): string {
  if (!dateStr) return "";
  // Strip trailing Z if present so browser parses as local time
  const safeStr = dateStr.endsWith("Z") ? dateStr.slice(0, -1) : dateStr;
  const d = new Date(safeStr);
  if (isNaN(d.getTime())) return dateStr;
  
  const pad = (n: number) => n.toString().padStart(2, '0');
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
