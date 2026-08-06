import { ArrowUpDown, HardDrive, Clock, } from "lucide-react";
import { useTranslation, } from "react-i18next";
import { Skeleton, } from "@/components/ui/skeleton";
import { Checkbox, } from "@/components/ui/checkbox";
import { TorrentRow, } from "@/components/torrent-row";
import type { Torrent, } from "@/lib/types";
import { cn, } from "@/lib/utils";
import React from "react";
import { LANG_NORMALIZE, } from "@/lib/constants";

const BADGE_COL_WIDTH: Record<string, string> = {
  en: "6rem",
  "zh-Hans": "4rem",
  ja: "8rem",
  ko: "6rem",
  fr: "9rem",
  it: "6.5rem",
  ru: "9.5rem",
  nl: "7rem",
};

interface TorrentListProps {
  torrents: Torrent[];
  isLoading: boolean;
  onMutate: () => void;
  onClickDetail: (t: Torrent,) => void;
  selectedIds: Set<string>;
  onToggleSelect: (infoHash: string,) => void;
  onToggleSelectAll: () => void;
}

export function TorrentList({
  torrents,
  isLoading,
  onMutate,
  onClickDetail,
  selectedIds,
  onToggleSelect,
  onToggleSelectAll,
}: TorrentListProps,) {
  const { t, i18n, } = useTranslation();
  const lang = LANG_NORMALIZE[i18n.language] ?? i18n.language;
  const badgeW = BADGE_COL_WIDTH[lang] ?? "6rem";
  const gridCols = `2.5rem ${badgeW} 1fr 6rem 4rem 5.5rem 5.5rem`;
  const allSelected = torrents.length > 0 && selectedIds.size === torrents.length;
  const someSelected = selectedIds.size > 0 && selectedIds.size < torrents.length;

  return (
    <div
      className="rounded-lg border border-border bg-card overflow-hidden"
      style={{ "--torrent-grid-cols": gridCols, } as React.CSSProperties}
    >
      <div
        className={cn(
          "grid items-center gap-x-3 border-b border-border bg-muted/50 px-3 py-2",
          torrents.length > 0 && "has-[[data-checked]]:bg-muted",
        )}
        style={{ gridTemplateColumns: gridCols, }}
      >
        <span className="flex items-center justify-center cursor-pointer">
          <Checkbox
            checked={allSelected}
            indeterminate={someSelected}
            onCheckedChange={onToggleSelectAll}
            aria-label={allSelected ? t("tooltip.deselectAll",) : t("tooltip.selectAll",)}
          />
        </span>
        <span />
        <span className="font-mono text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">{t("table.name",)}</span>
        <span className="hidden font-mono text-[10px] font-semibold uppercase tracking-widest text-muted-foreground text-right md:flex items-center justify-end gap-1">
          <HardDrive className="size-3" />
          {t("table.size",)}
        </span>
        <span className="hidden font-mono text-[10px] font-semibold uppercase tracking-widest text-muted-foreground text-right md:flex items-center justify-end gap-1">
          <ArrowUpDown className="size-3" />
          {t("table.seedersLeechers",)}
        </span>
        <span className="hidden font-mono text-[10px] font-semibold uppercase tracking-widest text-muted-foreground text-right md:flex items-center justify-end gap-1">
          <Clock className="size-3" />
          {t("table.published",)}
        </span>
        <span className="font-mono text-[10px] font-semibold uppercase tracking-widest text-muted-foreground text-right">
          {t("table.actions",)}
        </span>
      </div>

      {isLoading &&
        Array.from({ length: 10, },).map((_, i,) => (
          <div
            key={i}
            className="grid items-center gap-x-3 border-b border-border px-3 py-2.5 last:border-b-0"
            style={{ gridTemplateColumns: gridCols, }}
          >
            <Skeleton className="size-5 rounded" />
            <Skeleton className="h-5 rounded" style={{ width: badgeW, }} />
            <Skeleton className="h-3 w-full rounded" />
            <Skeleton className="hidden h-3 w-full rounded md:block" />
            <Skeleton className="hidden h-3 w-full rounded md:block" />
            <Skeleton className="hidden h-3 w-full rounded md:block" />
            <Skeleton className="h-7 w-full rounded" />
          </div>
        ),)}

      {!isLoading && torrents.length === 0 && (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <p className="font-mono text-3xl font-bold text-muted-foreground/30">{t("browse.noResultsTitle",)}</p>
          <p className="font-mono text-sm text-muted-foreground mt-3">{t("browse.noResultsMessage",)}</p>
        </div>
      )}

      {!isLoading &&
        torrents.map((torrent,) => (
          <TorrentRow
            key={torrent.id}
            torrent={torrent}
            selected={selectedIds.has(torrent.infoHash,)}
            onToggleSelect={() => onToggleSelect(torrent.infoHash,)}
            onMutate={onMutate}
            onClickDetail={onClickDetail}
          />
        ),)}
    </div>
  );
}
