export function formatBytes(bytes: number, locale?: string,): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB",];
  const i = Math.floor(Math.log(bytes,) / Math.log(k,),);
  const num = (bytes / Math.pow(k, i,)).toFixed(2,);
  const formatted = locale
    ? new Intl.NumberFormat(locale, { minimumFractionDigits: 2, maximumFractionDigits: 2, },).format(Number(num,),)
    : num;
  return `${formatted} ${sizes[i]}`;
}

import type { TFunction, } from "i18next";

export function formatRelative(iso: string, t?: TFunction, locale?: string,): string {
  const diff = Date.now() - new Date(iso,).getTime();
  const secs = Math.floor(diff / 1000,);
  if (secs < 60) {
    if (t) return t("time.secondsAgo", { count: secs, },);
    return `${secs}s ago`;
  }
  const mins = Math.floor(secs / 60,);
  if (mins < 60) {
    if (t) return t("time.minutesAgo", { count: mins, },);
    return `${mins}m ago`;
  }
  const hrs = Math.floor(mins / 60,);
  if (hrs < 24) {
    if (t) return t("time.hoursAgo", { count: hrs, },);
    return `${hrs}h ago`;
  }
  if (t) return t("time.daysAgo", { count: Math.floor(hrs / 24,), },);
  return new Date(iso,).toLocaleDateString(locale || "en",);
}

export function formatNumber(n: number,): string {
  return n.toLocaleString();
}
