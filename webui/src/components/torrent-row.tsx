import { useTranslation, } from "react-i18next";
import { Magnet, Download, } from "lucide-react";
import { toast, } from "sonner";
import { Button, } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger, } from "@/components/ui/tooltip";
import { CategoryBadge, } from "@/components/category-badge";
import { formatBytes, formatRelative, } from "@/lib/format";
import type { Torrent, } from "@/lib/types";
import { cn, } from "@/lib/utils";

import { Checkbox, } from "@/components/ui/checkbox";

interface TorrentRowProps {
  torrent: Torrent;
  onMutate: () => void;
  onClickDetail?: (torrent: Torrent,) => void;
  selected?: boolean;
  onToggleSelect?: () => void;
}

export function TorrentRow({ torrent, onClickDetail, selected = false, onToggleSelect, }: TorrentRowProps,) {
  const { t, i18n, } = useTranslation();

  function copyMagnet() {
    navigator.clipboard.writeText(torrent.magnetLink,);
    toast.success(t("toast.magnetCopied",),);
  }

  return (
    <div
      role="button"
      tabIndex={0}
      aria-selected={selected}
      aria-label={t("tooltip.viewDetails", { name: torrent.name, },)}
      className={cn(
        "group cursor-pointer border-b border-border transition-colors",
        "hover:bg-muted/40",
        selected && "bg-accent/30",
      )}
      onClick={() => onClickDetail?.(torrent,)}
      onKeyDown={(e,) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClickDetail?.(torrent,);
        }
      }}
    >
      <div className="grid items-center gap-x-3 px-3 py-2.5" style={{ gridTemplateColumns: "var(--torrent-grid-cols)", }}>
        <span
          className="flex items-center justify-center self-stretch w-full"
          onClick={(e,) => e.stopPropagation()}
          onKeyDown={(e,) => e.stopPropagation()}
        >
          <Checkbox checked={selected} onCheckedChange={onToggleSelect} aria-label={t("tooltip.selectTorrent", { name: torrent.name, },)} />
        </span>
        <CategoryBadge contentType={torrent.contentType} />

        <div className="min-w-0">
          <p
            className={cn("font-mono text-sm font-medium leading-snug text-foreground truncate",)}
            title={torrent.name}
          >
            {torrent.name}
          </p>
          <p className="font-mono text-[10px] text-muted-foreground truncate leading-relaxed">{torrent.infoHash}</p>
        </div>

        <p className="hidden font-mono text-xs text-green text-right md:block">{formatBytes(torrent.size, i18n.language,)}</p>

        <div className="hidden text-right md:block">
          <p className="font-mono text-xs text-cyan leading-snug">{torrent.seeders.toLocaleString()}</p>
          <p className="font-mono text-[10px] text-muted-foreground leading-snug">{torrent.leechers.toLocaleString()}</p>
        </div>

        <p className="hidden font-mono text-xs text-muted-foreground text-right md:block">{formatRelative(torrent.addedAt, t,)}</p>

        <div className="flex items-center justify-end gap-0.5" onClick={(e,) => e.stopPropagation()}>
          <Tooltip>
            <TooltipTrigger
              render={<Button variant="ghost" size="icon" className="size-7 text-muted-foreground hover:text-green" onClick={copyMagnet} />}
            >
              <Magnet className="size-3.5" />
              <span className="sr-only">{t("tooltip.copyMagnetLink",)}</span>
            </TooltipTrigger>
            <TooltipContent>{t("tooltip.copyMagnetLink",)}</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon"
                  className={cn(
                    "size-7 transition-colors",
                    torrent.hasFile ? "text-muted-foreground hover:text-cyan" : "text-muted-foreground/25 cursor-not-allowed",
                  )}
                  disabled={!torrent.hasFile}
                  onClick={(e,) => {
                    if (torrent.hasFile) {
                      e.stopPropagation();
                      const a = document.createElement("a",);
                      a.href = `/api/torrents/${torrent.infoHash}/download`;
                      a.download = "";
                      a.click();
                    }
                  }}
                />
              }
            >
              <Download className="size-3.5" />
              <span className="sr-only">{t("tooltip.downloadTorrent",)}</span>
            </TooltipTrigger>
            <TooltipContent>{torrent.hasFile ? t("tooltip.downloadTorrent",) : t("tooltip.noFileAvailable",)}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  );
}
