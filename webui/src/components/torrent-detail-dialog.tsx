import { useMemo, useState, type CSSProperties, } from "react";
import { useMutation, useQuery, } from "urql";
import { useTranslation, } from "react-i18next";
import {
  Magnet,
  Download,
  FileText,
  HardDrive,
  Hash,
  Calendar,
  Users,
  TrendingDown,
  ChevronRight,
  Folder,
  Loader2,
  RefreshCw,
} from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, } from "@/components/ui/dialog";
import { Button, } from "@/components/ui/button";
import { ScrollArea, } from "@/components/ui/scroll-area";
import { Skeleton, } from "@/components/ui/skeleton";
import { CategoryBadge, } from "@/components/category-badge";
import { formatBytes, formatRelative, } from "@/lib/format";
import { TorrentFilesDocument, TorrentReprocessDocument, } from "@/lib/graphql/generated/graphql";
import { buildFileTree, } from "@/lib/graphql/adapters";
import type { Torrent, TorrentFileTreeNode, } from "@/lib/types";
import { toast, } from "sonner";
import { cn, } from "@/lib/utils";

interface TorrentDetailDialogProps {
  torrent: Torrent | null;
  open: boolean;
  onOpenChange: (open: boolean,) => void;
}

function fileIcon(filename: string,): string {
  const ext = filename.split(".",).pop()?.toLowerCase() ?? "";
  if (["mkv", "mp4", "avi", "mov", "webm",].includes(ext,)) return "VID";
  if (["mp3", "flac", "aac", "ogg", "wav",].includes(ext,)) return "AUD";
  if (["jpg", "jpeg", "png", "raw", "xmp", "gif",].includes(ext,)) return "IMG";
  if (["pdf", "epub", "mobi",].includes(ext,)) return "DOC";
  if (["zip", "rar", "gz", "001", "7z",].includes(ext,)) return "ARC";
  if (["exe", "iso", "dll", "msi",].includes(ext,)) return "BIN";
  if (["nfo", "txt", "md", "log",].includes(ext,)) return "TXT";
  if (["srt", "ass", "vtt",].includes(ext,)) return "SUB";
  return "FIL";
}

function externalLinks(source?: string, id?: string, contentType?: string,): { url: string; label: string }[] {
  if (!source || !id) return [];
  switch (source) {
    case "tmdb":
      if (contentType === "movie") return [{ url: `https://www.themoviedb.org/movie/${id}`, label: "TMDB", },];
      if (contentType === "tv_show") return [{ url: `https://www.themoviedb.org/tv/${id}`, label: "TMDB", },];
      return [];
    case "imdb":
      return [{ url: `https://www.imdb.com/title/${id}`, label: "IMDb", },];
    case "tvdb":
      return [{ url: `https://thetvdb.com/dereferrer/${id}`, label: "TVDB", },];
    default:
      return [];
  }
}

const ICON_COLORS: Record<string, string> = {
  VID: "oklch(0.78 0.12 200)",
  AUD: "oklch(var(--green-raw, 0.72 0.19 158))",
  IMG: "oklch(0.78 0.14 75)",
  DOC: "oklch(0.72 0.12 280)",
  ARC: "oklch(0.72 0.10 50)",
  BIN: "oklch(0.62 0.22 25)",
  TXT: "oklch(0.60 0 0)",
  SUB: "oklch(0.72 0.18 320)",
  FIL: "oklch(0.55 0 0)",
};

export function TorrentDetailDialog({ torrent, open, onOpenChange, }: TorrentDetailDialogProps,) {
  const { t, i18n, } = useTranslation();
  const [result,] = useQuery({
    query: TorrentFilesDocument,
    variables: {
      input: { infoHashes: torrent ? [torrent.infoHash,] : [], },
    },
    pause: !open || !torrent,
  },);

  const rawFiles = result.data?.torrent?.files?.items;
  const treeData = useMemo(() => {
    if (rawFiles && rawFiles.length > 0) {
      return buildFileTree(rawFiles,);
    }
    return [];
  }, [rawFiles,],);

  const totalFiles = useMemo(() => {
    let count = 0;
    function walk(nodes: TorrentFileTreeNode[],) {
      for (const n of nodes) {
        if (n.type === "file") count++;
        walk(n.children,);
      }
    }
    walk(treeData,);
    return count;
  }, [treeData,],);

  const folderPaths = useMemo(() => {
    const paths = new Set<string>();
    function walk(nodes: TorrentFileTreeNode[],) {
      for (const n of nodes) {
        if (n.type === "folder") {
          paths.add(n.path,);
          walk(n.children,);
        }
      }
    }
    walk(treeData,);
    return paths;
  }, [treeData,],);

  const largestFile = useMemo(() => {
    function findMax(nodes: TorrentFileTreeNode[],): TorrentFileTreeNode | null {
      let best: TorrentFileTreeNode | null = null;
      for (const n of nodes) {
        if (n.type === "file" && (!best || n.size > best.size)) best = n;
        const childBest = findMax(n.children,);
        if (childBest && (!best || childBest.size > best.size)) best = childBest;
      }
      return best;
    }
    return findMax(treeData,);
  }, [treeData,],);

  const [collapsedPaths, setCollapsedPaths,] = useState<Set<string>>(new Set(),);

  const expanded = useMemo(() => {
    const result = new Set(folderPaths,);
    for (const p of collapsedPaths) result.delete(p,);
    return result;
  }, [folderPaths, collapsedPaths,],);

  function toggleFolder(path: string,) {
    setCollapsedPaths((prev,) => {
      const next = new Set(prev,);
      if (next.has(path,)) next.delete(path,);
      else next.add(path,);
      return next;
    },);
  }

  const allExpanded = expanded.size === folderPaths.size;

  function toggleAll() {
    setCollapsedPaths(allExpanded ? folderPaths : new Set(),);
  }

  const [reprocessState, reprocessTorrents,] = useMutation(TorrentReprocessDocument,);

  if (!torrent) return null;

  function copyMagnet() {
    navigator.clipboard.writeText(torrent!.magnetLink,);
    toast.success(t("toast.magnetCopied",),);
  }

  async function handleReprocess() {
    if (!torrent) return;
    const r = await reprocessTorrents({
      input: { infoHashes: [torrent.infoHash,], },
    },);
    if (r.error) {
      toast.error(t("toast.failed.reprocess", { message: r.error.message, },),);
      return;
    }
    toast.success(t("toast.reprocessingSingle",),);
  }

  function renderTree(nodes: TorrentFileTreeNode[], depth: number,) {
    return nodes.map((node,) => {
      if (node.type === "folder") {
        const isExpanded = expanded.has(node.path,);
        const pct = torrent!.size > 0 ? Math.round((node.size / torrent!.size) * 100,) : 0;
        return (
          <div key={node.path}>
            <div
              role="button"
              tabIndex={0}
              onClick={() => toggleFolder(node.path,)}
              onKeyDown={(e,) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggleFolder(node.path,); } }}
              className="grid grid-cols-[2.25rem_1fr] items-start gap-3 rounded-md px-3 py-2 hover:bg-muted/50 transition-colors pl-[calc(1rem+var(--tree-depth)*1.25rem)] sm:grid-cols-[2.25rem_1fr_10rem] sm:pl-[calc(1rem+var(--tree-depth)*3rem)]"
              style={{ "--tree-depth": depth, } as CSSProperties}
            >
              <div className="flex items-center justify-center gap-0.5 self-center">
                <ChevronRight className={`size-3.5 text-muted-foreground transition-transform duration-300 ${isExpanded ? "rotate-90" : ""}`} />
                <Folder className="size-5 text-muted-foreground" />
              </div>
              <div className="flex items-center gap-2 min-w-0 self-center">
                <span className="font-mono text-xs font-semibold text-foreground break-all">{node.name}</span>
                <span className="font-mono text-[10px] text-muted-foreground shrink-0">
                  {node.fileCount} {t("detail.files",)}
                </span>
              </div>
              <div className="hidden items-center gap-2 self-center justify-self-end sm:flex">
                <span className="font-mono text-xs text-foreground whitespace-nowrap">{formatBytes(node.size, i18n.language,)}</span>
                <div className="h-1 w-20 overflow-hidden rounded-full bg-secondary">
                  <div className="h-full rounded-full bg-muted-foreground/30" style={{ width: `${Math.max(pct, 1,)}%`, }} />
                </div>
              </div>
            </div>
            <div className={`grid transition-all duration-300 ease-in-out ${isExpanded ? "grid-rows-[1fr] opacity-100 visible" : "grid-rows-[0fr] opacity-0 invisible"}`}>
              <div className="overflow-hidden min-w-0">
                {renderTree(node.children, depth + 1,)}
              </div>
            </div>
          </div>
        );
      }
      const type = fileIcon(node.name,);
      const color = ICON_COLORS[type] ?? "oklch(0.55 0 0)";
      const pct = torrent!.size > 0 ? Math.round((node.size / torrent!.size) * 100,) : 0;
      return (
        <div
          key={node.path}
          className="grid grid-cols-[2.25rem_1fr] items-start gap-3 rounded-md px-3 py-2 hover:bg-muted/50 transition-colors pl-[calc(1rem+var(--tree-depth)*1.25rem)] sm:grid-cols-[2.25rem_1fr_10rem] sm:pl-[calc(1rem+var(--tree-depth)*3rem)]"
          style={{ "--tree-depth": depth, } as CSSProperties}
        >
          <span
            className="flex items-center justify-center rounded py-0.5 font-mono text-[10px] font-bold text-background self-center"
            style={{ backgroundColor: color, }}
          >
            {type}
          </span>
          <p className="font-mono text-xs break-all min-w-0">
            {node.name ? (
              <span className="text-foreground">{node.name}</span>
            ) : (
              <span className="text-muted-foreground italic">&lt;{t("detail.itemWithoutName",)}&gt;</span>
            )}
          </p>
          <div className="hidden items-center gap-2 self-center justify-self-end sm:flex">
            <span className="font-mono text-xs text-foreground whitespace-nowrap">{formatBytes(node.size,)}</span>
            <div className="h-1 w-20 overflow-hidden rounded-full bg-secondary">
              <div className="h-full rounded-full" style={{ width: `${Math.max(pct, 2,)}%`, backgroundColor: color, opacity: 0.7, }} />
            </div>
          </div>
        </div>
      );
    },);
  }

  const links = externalLinks(torrent.contentSource, torrent.contentId, torrent.contentType,);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!max-w-4xl gap-0 p-0 overflow-hidden max-h-[calc(100dvh-1rem)] md:max-h-[calc(100dvh-2rem)] max-md:flex max-md:flex-col data-open:[--tw-enter-translate-y:1rem] data-open:[--tw-animate-duration:300ms] data-closed:[--tw-exit-translate-y:1rem] data-closed:[--tw-animate-duration:200ms]">
        <DialogHeader className="shrink-0 p-4 pb-4 border-b border-border sm:p-6 sm:pb-4">
          <div className="grid grid-cols-[auto_1fr_0rem] items-start gap-3">
            <CategoryBadge contentType={torrent.contentType} className="mt-0.5" />
            <div className="min-w-0">
              <DialogTitle className="font-mono text-sm font-semibold text-foreground leading-snug text-left break-all">
                {torrent.name}
              </DialogTitle>
              <DialogDescription className="font-mono text-xs text-muted-foreground mt-1 text-left break-all">
                {torrent.infoHash}
              </DialogDescription>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-px bg-border sm:grid-cols-4">
            {[
              {
                icon: HardDrive,
                labelKey: "detail.totalSize",
                value: formatBytes(torrent!.size, i18n.language,),
                accent: true,
              },
              {
                icon: FileText,
                labelKey: "detail.files",
                value: totalFiles.toString(),
                accent: false,
              },
              {
                icon: Users,
                labelKey: "detail.seeders",
                value: torrent.seeders.toLocaleString(),
                accent: false,
              },
              {
                icon: TrendingDown,
                labelKey: "detail.leechers",
                value: torrent.leechers.toLocaleString(),
                accent: false,
              },
            ].map(({ icon: Icon, labelKey, value, accent, },) => (
              <div key={labelKey} className="flex flex-col gap-1 bg-card p-4">
                <div className="flex items-center gap-1.5">
                  <Icon className="size-3.5 text-muted-foreground" />
                  <span className="font-mono text-xs text-muted-foreground">{t(labelKey,)}</span>
                </div>
                <span className={cn("font-mono text-lg font-bold", accent ? "text-green" : "text-foreground",)}>{value}</span>
              </div>
            ),)}
          </div>

          <div className="flex flex-wrap gap-x-6 gap-y-2 px-4 pt-2">
            <div className="flex items-center gap-1.5">
              <Hash className="size-3 text-muted-foreground" />
              <span className="font-mono text-xs text-muted-foreground">{t("detail.added",)}</span>
              <span className="font-mono text-xs text-foreground">{formatRelative(torrent.addedAt, t,)}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Calendar className="size-3 text-muted-foreground" />
              <span className="font-mono text-xs text-muted-foreground">{t("detail.date",)}</span>
              <span className="font-mono text-xs text-foreground">{new Date(torrent.addedAt,).toLocaleDateString(i18n.language,)}</span>
            </div>
            {largestFile && totalFiles > 1 && (
              <div className="flex items-center gap-1.5">
                <Folder className="size-3 text-muted-foreground" />
                <span className="font-mono text-xs text-muted-foreground">{t("detail.largest",)}</span>
                <span className="font-mono text-xs text-foreground truncate max-w-48">
                  {largestFile.name} ({formatBytes(largestFile.size, i18n.language,)})
                </span>
              </div>
            )}
          </div>

        </DialogHeader>

        <ScrollArea className="h-[45dvh] min-h-0 flex-1 md:h-[55vh] md:min-h-[300px]">
          <div className="px-3 py-4 sm:px-6">
            <div className="flex items-center justify-between mb-3 py-1">
              <p className="font-mono text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-2">
                {t("detail.fileList", { count: totalFiles, },)}
                {result.fetching && <Loader2 className="size-3 animate-spin" />}
              </p>
              {folderPaths.size > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="font-mono text-xs font-normal text-muted-foreground hover:text-cyan"
                  onClick={toggleAll}
                >
                  {allExpanded ? t("detail.collapseAll",) : t("detail.expandAll",)}
                </Button>
              )}
            </div>
            {totalFiles === 0 && result.fetching ? (
              <div className="flex flex-col gap-2">
                {Array.from({ length: 6, },).map((_, i,) => (
                  <div key={i} className="flex items-center gap-3 rounded-md px-3 py-2">
                    <Skeleton className="h-5 w-9 shrink-0 rounded" />
                    <div className="min-w-0 flex-1 flex flex-col gap-1.5">
                      <Skeleton className="h-3 w-3/5 rounded" />
                      <Skeleton className="h-2 w-2/5 rounded" />
                    </div>
                    <div className="flex shrink-0 flex-col items-end gap-1.5">
                      <Skeleton className="h-3 w-14 rounded" />
                      <Skeleton className="h-1.5 w-20 rounded-full" />
                    </div>
                  </div>
                ),)}
              </div>
            ) : totalFiles === 0 && !result.fetching ? (
              <div className="flex h-full min-h-[150px] items-center justify-center">
                <p className="font-mono text-sm text-muted-foreground">{t("browse.noResultsMessage",)}</p>
              </div>
            ) : (
              <div className="flex flex-col gap-1">{renderTree(treeData, 0,)}</div>
            )}
          </div>
        </ScrollArea>

        <div className="flex shrink-0 flex-wrap items-center gap-2 border-t border-border px-4 py-4 sm:px-6">
          <Button
            variant="default"
            size="sm"
            className="gap-1.5 font-mono text-xs bg-green/20 text-green border border-green/40 hover:bg-green/30"
            onClick={copyMagnet}
          >
            <Magnet className="size-3.5" />
            {t("detail.copyMagnet",)}
          </Button>

          {torrent.hasFile && (
            <a
              href={`/api/torrents/${torrent.infoHash}/download`}
              download
              className="inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-3 py-1.5 font-mono text-xs text-foreground transition-colors hover:bg-muted active:translate-y-px"
            >
              <Download className="size-3.5" />
              {t("detail.downloadTorrent",)}
            </a>
          )}

          <Button
            variant="ghost"
            size="sm"
            className="gap-1.5 font-mono text-xs text-muted-foreground hover:text-cyan"
            onClick={handleReprocess}
            disabled={reprocessState.fetching}
          >
            <RefreshCw className={`size-3.5 ${reprocessState.fetching ? "animate-spin" : ""}`} />
            {t("detail.reprocess",)}
          </Button>

          {links.length > 0 && (
            <div className="ml-auto flex items-center gap-1.5">
              {links.map((link,) => (
                <a
                  key={link.label}
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 rounded-md border border-border bg-background px-2 py-1 font-mono text-xs text-muted-foreground transition-colors hover:text-foreground hover:border-foreground/30"
                >
                  {link.label}
                </a>
              ),)}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
