import type { Torrent, TorrentCategory, TorrentFileTreeNode, } from "@/lib/types";

interface TorrentSearchItem {
  infoHash: string;
  contentType?: string | null;
  contentSource?: string | null;
  contentId?: string | null;
  title: string;
  torrent?: {
    infoHash: string;
    name: string;
    size: number;
    filesCount?: number | null;
    hasFilesInfo?: boolean | null;
    fileType?: string | null;
    seeders?: number | null;
    leechers?: number | null;
    magnetUri: string;
  } | null;
  seeders?: number | null;
  leechers?: number | null;
  createdAt: string;
}

const CATEGORY_MAP: Record<string, TorrentCategory> = {
  movie: "video",
  tv_show: "video",
  music: "audio",
  ebook: "books",
  comic: "images",
  audiobook: "audio",
  game: "games",
  software: "software",
  adult: "adult",
  other: "other",
};

function mapCategory(contentType?: string | null,): TorrentCategory {
  if (!contentType) return "other";
  return CATEGORY_MAP[contentType] || "other";
}

export function toTorrent(item: TorrentSearchItem,): Torrent {
  const t = item.torrent;
  const infoHash = String(item.infoHash ?? "",);
  return {
    id: infoHash,
    infoHash,
    name: item.title || t?.name || "Unknown",
    size: t?.size ?? 0,
    category: mapCategory(item.contentType,),
    contentType: item.contentType ?? undefined,
    contentSource: item.contentSource ?? undefined,
    contentId: item.contentId ?? undefined,
    seeders: item.seeders ?? t?.seeders ?? 0,
    leechers: item.leechers ?? t?.leechers ?? 0,
    addedAt: String(item.createdAt ?? "",),
    magnetLink: t?.magnetUri ?? `magnet:?xt=urn:btih:${infoHash}`,
    hasFile: t?.hasFilesInfo === true,
  };
}

function treeNodeSorter(a: TorrentFileTreeNode, b: TorrentFileTreeNode,): number {
  if (a.type === "folder" && b.type !== "folder") return -1;
  if (a.type !== "folder" && b.type === "folder") return 1;
  return a.name.toLowerCase().localeCompare(b.name.toLowerCase(),);
}

export function buildFileTree(files: { pathParts: string[]; size: number; index?: number }[],): TorrentFileTreeNode[] {
  interface TrieNode {
    name: string;
    path: string;
    size: number;
    fileCount: number;
    fileType?: string | null;
    children: Map<string, TrieNode>;
  }

  const root = new Map<string, TrieNode>();

  for (const file of files) {
    const parts = file.pathParts;

    if (parts.length === 0) {
      const key = `__unnamed_${file.index ?? root.size}`;
      root.set(key, { name: "", path: "", size: file.size, fileCount: 1, children: new Map(), },);
      continue;
    }

    let current = root;
    let pathAcc = "";

    for (let i = 0; i < parts.length - 1; i++) {
      const seg = parts[i];
      pathAcc = pathAcc ? `${pathAcc}/${seg}` : seg;
      if (!current.has(seg,)) {
        current.set(seg, {
          name: seg,
          path: pathAcc,
          size: 0,
          fileCount: 0,
          children: new Map(),
        },);
      }
      current = current.get(seg,)!.children;
    }

    const fileName = parts[parts.length - 1];
    if (!current.has(fileName,)) {
      current.set(fileName, {
        name: fileName,
        path: file.pathParts.join("/",),
        size: file.size,
        fileCount: 1,
        children: new Map(),
      },);
    }
  }

  function flatten(trie: Map<string, TrieNode>,): TorrentFileTreeNode[] {
    const nodes: TorrentFileTreeNode[] = [];
    for (const [, node,] of trie) {
      const children = flatten(node.children,);
      let size = node.size;
      let fileCount = node.fileCount;
      for (const child of children) {
        size += child.size;
        fileCount += child.fileCount;
      }
      nodes.push({
        name: node.name,
        path: node.path,
        size,
        fileCount,
        type: children.length > 0 ? "folder" : "file",
        fileType: node.fileType ?? undefined,
        children,
      },);
    }
    nodes.sort(treeNodeSorter,);
    return nodes;
  }

  return flatten(root,);
}
