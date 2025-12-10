import { format, formatDistance } from "date-fns";
import { ru } from "date-fns/locale";

export function formatDate(date: string | Date): string {
  return format(new Date(date), "dd.MM.yyyy", { locale: ru });
}

export function formatDateTime(date: string | Date): string {
  return format(new Date(date), "dd.MM.yyyy HH:mm:ss", { locale: ru });
}

export function formatRelative(date: string | Date): string {
  return formatDistance(new Date(date), new Date(), {
    addSuffix: true,
    locale: ru,
  });
}

export function formatNumber(num: number, decimals = 2): string {
  return new Intl.NumberFormat("ru-RU", {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(num);
}

export function formatCurrency(amount: number, currency = "RUB"): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency,
  }).format(amount);
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms} мс`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)} сек`;
  return `${(ms / 60000).toFixed(1)} мин`;
}
