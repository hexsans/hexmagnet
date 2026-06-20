export type TorrentCategory = "video" | "audio" | "software" | "games" | "books" | "images" | "archives" | "adult" | "other";

export interface TorrentFileTreeNode {
  name: string;
  path: string;
  size: number;
  fileCount: number;
  type: "folder" | "file";
  fileType?: string | null;
  children: TorrentFileTreeNode[];
}

export interface Torrent {
  id: string;
  infoHash: string;
  name: string;
  size: number;
  category: TorrentCategory;
  contentType?: string;
  contentSource?: string;
  contentId?: string;
  seeders: number;
  leechers: number;
  addedAt: string;
  magnetLink: string;
  hasFile: boolean;
}

export interface CrawlerStatus {
  running: boolean;
  torrentsCrawled: number;
  peersConnected: number;
  peersDiscovered: number;
  recentActivity: { id: string; type: string; message: string; time: string }[];
}


