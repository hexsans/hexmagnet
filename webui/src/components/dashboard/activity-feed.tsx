import { useTranslation, } from "react-i18next";
import { ScrollArea, } from "@/components/ui/scroll-area";
import { formatRelative, } from "@/lib/format";
import { cn, } from "@/lib/utils";
import type { CrawlerStatus, } from "@/lib/types";

const TYPE_STYLES: Record<string, { dotColor: string; text: string; label: string }> = {
  discovered: {
    dotColor: "oklch(0.72 0.19 158)",
    text: "text-green",
    label: "PEER",
  },
  metadata: {
    dotColor: "oklch(0.78 0.12 200)",
    text: "text-cyan",
    label: "META",
  },
  peer: {
    dotColor: "oklch(0.78 0.14 75)",
    text: "text-amber",
    label: "NODE",
  },
  error: {
    dotColor: "oklch(0.62 0.22 25)",
    text: "text-destructive",
    label: "ERR ",
  },
} as const;

interface ActivityFeedProps {
  activity: CrawlerStatus["recentActivity"];
}

export function ActivityFeed({ activity, }: ActivityFeedProps,) {
  const { t, } = useTranslation();
  return (
    <ScrollArea className="h-72">
      <div className="flex flex-col gap-0.5 pr-3">
        {activity.map((item,) => {
          const s = TYPE_STYLES[item.type] ?? { dotColor: "oklch(0.5 0 0)", text: "text-muted-foreground", label: "UNKN", };
          return (
            <div key={item.id} className="flex items-start gap-2 py-1 border-b border-border/40 last:border-0">
              <span className="mt-1.5 size-1.5 shrink-0 rounded-full" style={{ backgroundColor: s.dotColor, }} />
              <span className={cn("font-mono text-xs shrink-0 w-9", s.text,)}>{s.label}</span>
              <span className="font-mono text-xs text-foreground flex-1 leading-relaxed break-words [font-variant-ligatures:none]">
                {item.message}
              </span>
              <span className="font-mono text-xs text-muted-foreground shrink-0">{formatRelative(item.time, t,)}</span>
            </div>
          );
        },)}
      </div>
    </ScrollArea>
  );
}
