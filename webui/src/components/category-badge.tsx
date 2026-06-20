import { useTranslation, } from "react-i18next";
import { Badge, } from "@/components/ui/badge";
import type { TorrentCategory, } from "@/lib/types";
import { cn, } from "@/lib/utils";

const CATEGORY_CONFIG: Record<TorrentCategory, { i18nKey: string; className: string }> = {
  video: { i18nKey: "category.video", className: "bg-cyan/10 text-cyan border-cyan/30", },
  audio: { i18nKey: "category.audio", className: "bg-green/10 text-green border-green/30", },
  software: { i18nKey: "category.software", className: "bg-amber/10 text-amber border-amber/30", },
  games: { i18nKey: "category.games", className: "bg-[oklch(0.62_0.22_25_/_0.10)] text-[oklch(0.72_0.22_25)] border-[oklch(0.62_0.22_25_/_0.30)]", },
  books: { i18nKey: "category.books", className: "bg-[oklch(0.52_0.12_280_/_0.10)] text-[oklch(0.72_0.12_280)] border-[oklch(0.52_0.12_280_/_0.30)]", },
  images: { i18nKey: "category.images", className: "bg-[oklch(0.65_0.18_320_/_0.10)] text-[oklch(0.72_0.18_320)] border-[oklch(0.65_0.18_320_/_0.30)]", },
  archives: { i18nKey: "category.archives", className: "bg-[oklch(0.55_0.10_50_/_0.10)] text-[oklch(0.72_0.10_50)] border-[oklch(0.55_0.10_50_/_0.30)]", },
  adult: { i18nKey: "category.adult", className: "bg-[oklch(0.55_0.22_320_/_0.10)] text-[oklch(0.75_0.22_320)] border-[oklch(0.55_0.22_320_/_0.30)]", },
  other: { i18nKey: "category.other", className: "bg-muted text-muted-foreground border-border", },
};

const CONTENT_TYPE_CONFIG: Record<string, { i18nKey: string; className: string }> = {
  movie: { i18nKey: "category.movie", className: "bg-cyan/10 text-cyan border-cyan/30", },
  tv_show: { i18nKey: "category.tvShow", className: "bg-sky/10 text-sky border-sky/30", },
  music: { i18nKey: "category.music", className: "bg-green/10 text-green border-green/30", },
  ebook: { i18nKey: "category.ebook", className: "bg-violet/10 text-violet border-violet/30", },
  comic: { i18nKey: "category.comic", className: "bg-pink/10 text-pink border-pink/30", },
  audiobook: { i18nKey: "category.audiobook", className: "bg-emerald/10 text-emerald border-emerald/30", },
  game: { i18nKey: "category.game", className: "bg-orange/10 text-orange border-orange/30", },
  software: { i18nKey: "category.software", className: "bg-amber/10 text-amber border-amber/30", },
  adult: { i18nKey: "category.adult", className: "bg-fuchsia/10 text-fuchsia border-fuchsia/30", },
  unknown: { i18nKey: "category.unknown", className: "bg-muted text-muted-foreground border-border", },
  other: { i18nKey: "category.other", className: "bg-muted text-muted-foreground border-border", },
};

interface CategoryBadgeProps {
  category?: TorrentCategory;
  contentType?: string;
  className?: string;
}

export function CategoryBadge({ category, contentType, className, }: CategoryBadgeProps,) {
  const { t, } = useTranslation();
  let config;

  if (contentType && CONTENT_TYPE_CONFIG[contentType]) {
    config = CONTENT_TYPE_CONFIG[contentType];
  } else if (category && CATEGORY_CONFIG[category]) {
    config = CATEGORY_CONFIG[category];
  } else {
    config = CATEGORY_CONFIG.other;
  }

  return (
    <Badge variant="outline" className={cn("font-mono text-xs shrink-0", config.className, className,)}>
      {t(config.i18nKey,)}
    </Badge>
  );
}
