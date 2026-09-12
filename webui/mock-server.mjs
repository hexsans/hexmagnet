import http from "http";

const PORT = process.env.MOCK_PORT || 4000;
const API_PREFIX = "/api";

const CORS = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type",
};

const CATEGORIES = [
  "movie",
  "tv_show",
  "music",
  "ebook",
  "comic",
  "audiobook",
  "game",
  "software",
  "other",
];
const SOURCES = ["dht"];

const MOVIE_TITLES = [
  "The Matrix 1999 2160p BluRay REMUX HEVC DTS-HD MA TrueHD 7 1 Atmos",
  "Inception 2010 1080p BluRay x264 DTS 5 1",
  "Interstellar 2014 2160p WEB-DL DD 5 1 H 264",
  "Parasite 2019 1080p BluRay x264 AC3",
  "Everything Everywhere All At Once 2022 1080p WEBRip x264",
  "Blade Runner 2049 2017 2160p BluRay REMUX HEVC Atmos",
  "The Shawshank Redemption 1994 1080p BluRay x264",
  "Pulp Fiction 1994 1080p WEB-DL x264 AAC",
  "Dune Part Two 2024 2160p WEB-DL H 265",
  "Spirited Away 2001 1080p BluRay FLAC",
  "The Ultimate Extended Director's Cut Collector's Edition with All Available Bonus Features and Behind the Scenes Content 2024 2160p UltraHD BluRay REMUX HEVC DTS-HD MA TrueHD 7 1 Atmos HDR10+ Dolby Vision",
];

const TV_TITLES = [
  "Breaking Bad S01E01 1080p WEB-DL DD 5 1 H 264",
  "Game of Thrones S08E03 1080p WEBRip x265",
  "Stranger Things S04 Complete 2160p WEB-DL DV HDR",
  "The Last of Us S01 Complete 1080p BluRay x264",
  "Better Call Saul S06 Complete 1080p WEB-DL",
  "Succession S04E01 1080p WEB-DL H 264 AAC",
  "The Bear S02 Complete 1080p WEBRip x265",
  "Severance S01 Complete 1080p WEB-DL DD 5 1",
  "House of the Dragon S01 2160p WEB-DL HDR",
  "Dark S03 Complete 1080p WEB-DL x264 AC3",
  "The Complete Series Ultimate Collection Bonus Discs Including All Extended Episodes and Special Features 1080p BluRay x264 DTS 5 1",
];

const MUSIC_TITLES = [
  "Various Artists - 2023 - Greatest Hits FLAC",
  "Daft Punk - 2013 - Random Access Memories 24bit FLAC",
  "Radiohead - 1997 - OK Computer FLAC",
  "Pink Floyd - 1973 - The Dark Side of the Moon 192kHz",
  "Miles Davis - 1959 - Kind of Blue DSD",
];

const EBOOK_TITLES = [
  "Richard Dawkins - The Selfish Gene epub",
  "Yuval Noah Harari - Sapiens A Brief History of Humankind pdf",
  "Robert C Martin - Clean Code epub",
  "Stephen Hawking - A Brief History of Time mobi",
  "Thomas H Cormen - Introduction to Algorithms pdf",
];

const GAME_TITLES = [
  "Elden Ring 2022 Deluxe Edition GOG",
  "Baldurs Gate 3 2023 Complete Edition ISO",
  "The Witcher 3 Wild Hunt Complete Edition GOG",
  "Red Dead Redemption 2 2019 repack",
  "Cyberpunk 2077 Phantom Liberty GOG",
];

const FILES_BY_TYPE = {
  movie: [
    { path: "video/movie.mkv", type: "video", size: [5e9, 15e9, 30e9] },
    { path: "video/featurettes/making-of.mkv", type: "video", size: [1e9, 2e9, 4e9] },
    { path: "video/featurettes/deleted-scenes.mkv", type: "video", size: [5e8, 1e9, 2e9] },
    { path: "subs/english.srt", type: "subtitles", size: [5e4, 1e5, 2e5] },
    { path: "subs/french.srt", type: "subtitles", size: [5e4, 1e5, 2e5] },
    { path: "subs/spanish.srt", type: "subtitles", size: [5e4, 1e5, 2e5] },
    { path: "images/poster.jpg", type: "image", size: [2e5, 5e5, 1e6] },
    { path: "images/backdrops/01.jpg", type: "image", size: [1e6, 2e6, 5e6] },
    { path: "images/backdrops/02.jpg", type: "image", size: [1e6, 2e6, 5e6] },
    { path: "images/backdrops/03.jpg", type: "image", size: [1e6, 2e6, 5e6] },
    { path: "docs/movie.nfo", type: "document", size: [1e3, 5e3, 1e4] },
    { path: "Bonus Features/Behind the Scenes/The Making Of Documentary Featurette Extended Version with Director Commentary.mkv", type: "video", size: [2e9, 4e9, 8e9] },
    { path: "Extras/Deleted Scenes/Alternate Ending Director's Cut Extended Version with Optional Director Commentary Track.mkv", type: "video", size: [1e9, 2e9, 4e9] },
    { path: "A Very Long Directory Name That Will Definitely Overflow The Container And Force The Name To Wrap To Multiple Lines When Viewed In The Torrent Detail Dialog/video.mkv", type: "video", size: [2e9, 5e9, 8e9] },
  ],
  tv_show: [
    { path: "Season 1/Episodes/S01E01.mkv", type: "video", size: [1e9, 2e9, 3e9] },
    { path: "Season 1/Episodes/S01E02.mkv", type: "video", size: [1e9, 2e9, 3e9] },
    { path: "Season 1/Episodes/S01E03.mkv", type: "video", size: [1e9, 2e9, 3e9] },
    { path: "Season 1/Subs/English/english.srt", type: "subtitles", size: [3e4, 5e4, 8e4] },
    { path: "Season 1/Subs/French/french.srt", type: "subtitles", size: [3e4, 5e4, 8e4] },
    { path: "Season 1/Thumbnails/episode-01.jpg", type: "image", size: [1e5, 2e5, 5e5] },
    { path: "Season 1/Thumbnails/episode-02.jpg", type: "image", size: [1e5, 2e5, 5e5] },
    { path: "Season 1/Thumbnails/episode-03.jpg", type: "image", size: [1e5, 2e5, 5e5] },
    { path: "Season 2/Episodes/S02E01.mkv", type: "video", size: [1e9, 2e9, 3e9] },
    { path: "Season 2/Episodes/S02E02.mkv", type: "video", size: [1e9, 2e9, 3e9] },
    { path: "Season 2/Subs/english.srt", type: "subtitles", size: [3e4, 5e4, 8e4] },
    { path: "Season 2/Thumbnails/episode-01.jpg", type: "image", size: [1e5, 2e5, 5e5] },
    { path: "Specials/behind-the-scenes.mkv", type: "video", size: [2e9, 4e9, 6e9] },
    { path: "Specials/featurette.mkv", type: "video", size: [1e9, 3e9, 5e9] },
    { path: "Season 1/Extras/Bonus Features/Behind the Scenes/Making of Season 1 Documentary with Cast and Crew Interviews.mkv", type: "video", size: [3e9, 5e9, 8e9] },
  ],
  music: [
    { path: "tracks/01-track.flac", type: "audio", size: [2e7, 5e7, 1e8] },
    { path: "tracks/02-track.flac", type: "audio", size: [2e7, 5e7, 1e8] },
    { path: "cover.jpg", type: "image", size: [2e5, 5e5, 1e6] },
    { path: "album.nfo", type: "document", size: [1e3, 2e3, 5e3] },
  ],
  ebook: [
    { path: "book.epub", type: "document", size: [1e6, 5e6, 2e7] },
    { path: "book.pdf", type: "document", size: [5e6, 2e7, 5e7] },
    { path: "cover.jpg", type: "image", size: [1e5, 3e5, 8e5] },
  ],
  software: [
    { path: "setup/setup.exe", type: "software", size: [5e7, 2e8, 5e8] },
    { path: "setup/setup.msi", type: "software", size: [3e7, 1e8, 3e8] },
    { path: "crack.dll", type: "data", size: [1e5, 5e5, 1e6] },
    { path: "readme.txt", type: "document", size: [1e3, 5e3, 1e4] },
  ],
  game: [
    { path: "discs/game.iso", type: "data", size: [4e9, 1e10, 5e10] },
    { path: "discs/game.bin", type: "data", size: [2e9, 5e9, 2e10] },
    { path: "crack/crack.exe", type: "software", size: [1e6, 5e6, 1e7] },
    { path: "setup/setup.exe", type: "software", size: [5e7, 2e8, 5e8] },
  ],
  audiobook: [
    { path: "tracks/part01.mp3", type: "audio", size: [3e7, 8e7, 2e8] },
    { path: "tracks/part02.mp3", type: "audio", size: [3e7, 8e7, 2e8] },
    { path: "cover.jpg", type: "image", size: [1e5, 3e5, 8e5] },
    { path: "chapters.nfo", type: "document", size: [1e3, 5e3, 1e4] },
  ],
  comic: [
    { path: "issue-001.cbr", type: "image", size: [2e7, 5e7, 1e8] },
    { path: "issue-002.cbr", type: "image", size: [2e7, 5e7, 1e8] },
    { path: "cover.jpg", type: "image", size: [3e5, 8e5, 2e6] },
  ],
  other: [
    { path: "video.mp4", type: "video", size: [5e8, 2e9, 5e9] },
    { path: "video-subtitles.srt", type: "subtitles", size: [2e4, 5e4, 8e4] },
    { path: "preview.jpg", type: "image", size: [2e5, 5e5, 1e6] },
  ],
};

const FILE_STATUSES = ["single", "multi", "over_threshold"];
const LANGUAGES = [
  { id: "en", name: "English" },
  { id: "ja", name: "Japanese" },
  { id: "fr", name: "French" },
  { id: "de", name: "German" },
  { id: "es", name: "Spanish" },
  { id: "ko", name: "Korean" },
];

function choose(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function makeInfoHash(index) {
  const hex = index.toString(16).padStart(2, "0");
  return `abcdef0123456789abcdef0123456789abcdef${hex}`;
}

function makeMagnet(hash, name) {
  const encoded = encodeURIComponent(name);
  return `magnet:?xt=urn:btih:${hash}&dn=${encoded}&tr=udp://tracker.opentrackr.org:1337/announce`;
}

function pickFiles(type, infoHash, output) {
  const templates = FILES_BY_TYPE[type] || FILES_BY_TYPE.movie;
  const files = templates.map((t, i) => {
    const path = typeof t.path === "function" ? t.path(i) : t.path;
    return {
      infoHash,
      index: i,
      path,
      pathParts: path.split("/"),
      size: choose(t.size),
      fileType: t.type,
    };
  });
  output.push(...files);
  return files;
}

let torrentIdx = 0;
const allFiles = [];
let mockReindex = { total: 0, indexed: 0, done: true, running: false, error: null };
let mockReclassify = { total: 0, processed: 0, done: true, running: false, error: null };

const DEFAULT_CONFIG = {
  dht: {
    port: 3334,
    responder: { enabled: true, globalRateLimit: 50, perIPRateLimit: 1 },
    bootstrapNodes: [
      "router.bittorrent.com:6881",
      "dht.transmissionbt.com:6881",
      "dht.aelitis.com:6881",
    ],
    reseedBootstrapNodesInterval: 60,
    requester: {
      requestLimit: 50,
      rescrapeThreshold: 2592000,
      hashDiscoverLimit: 10,
    },
  },
  server: {
    ip: "0.0.0.0",
    port: 3333,
    embedTrackers: [],
    torrentFilePath: "/app/torrents",
    log: {
      consoleLevel: "info",
      fileOutputLevel: "off",
      fileRotator: { path: "./logs", maxBackups: 7, format: "text" },
    },
  },
  classifier: {
    workflow: "",
    concurrency: 1,
    llm: {
      endpoint: "https://api.openai.com/v1",
      apiKey: "********************************",
      model: "gpt-4o-mini",
      timeout: 30,
      maxRetries: 2,
      temperature: 0,
      reasoningEffort: "",
      maxFiles: -1,
      prompt: "",
      enabled: false,
    },
    torrentFilter: { mode: "off", titlePatterns: [], filenamePatterns: [] },
    tmdb: { enabled: false, accessToken: "******************************e3f2", rateLimit: 20 },
  },
  storage: {
    postgres: {
      host: "localhost",
      username: "postgres",
      port: 5432,
      database: "hexmagnet",
      password: "****",
      sslMode: "disable",
      connectionTimeout: 0,
      sslCertPath: "/etc/ssl/certs/postgresql/client.crt",
      sslKeyPath: "/etc/ssl/private/postgresql/client.key",
      sslRootCertPath: "/etc/ssl/certs/ca-certificates.crt",
      maxConnections: 25,
    },
    search: { backend: "postgresql", elasticsearch: { addresses: ["http://localhost:9200"], embedding: { endpoint: "http://localhost:11434/v1", apikey: "ollama", model: "bge-m3", dimensions: 1024, instructionEnabled: false } } },
    queue: { backend: "memory", kafka: { brokers: ["localhost:9092"] } },
  },
};
let mockConfig = structuredClone(DEFAULT_CONFIG);

function generateTorrents(count) {
  const torrents = [];
  for (let i = 0; i < count; i++) {
    const idx = torrentIdx++;
    const infoHash = makeInfoHash(idx);
    const category = CATEGORIES[idx % CATEGORIES.length];
    const titlePool = MOVIE_TITLES;
    const name = `${choose(titlePool)}-${idx}`;
    const createdAt = new Date(
      Date.now() - randInt(0, 30) * 86400000,
    ).toISOString();
    const seeders = randInt(0, 500);
    const leechers = randInt(0, 100);
    const fileTemplate = FILES_BY_TYPE[category] || FILES_BY_TYPE.movie;
    let size = 0;
    const files = fileTemplate.map((t, fi) => {
      const fs = choose(t.size);
      size += fs;
      const path = typeof t.path === "function" ? t.path(fi) : t.path;
      return {
        infoHash,
        index: fi,
        path,
        pathParts: path.split("/"),
        size: fs,
        fileType: t.type,
      };
    });
    allFiles.push(...files);
    const filesCount = files.length;
    const filesStatus =
      filesCount > 15 ? "over_threshold" : filesCount > 1 ? "multi" : "single";

    torrents.push({
      id: String(idx + 1),
      infoHash,
      contentType: category,
      title: name,
      torrent: {
        infoHash,
        name,
        size,
        filesStatus,
        filesCount,
        hasFilesInfo: true,
        fileType: "video",
        sources: SOURCES.map((k) => ({
          key: k,
          name: "DHT",
        })),
        seeders,
        leechers,
        magnetUri: makeMagnet(infoHash, name),
      },
      seeders,
      leechers,
      createdAt,
      languages: [choose(LANGUAGES)],
      content:
        category === "movie" || category === "tv_show"
          ? {
              type: category,
              title: name.replace(/ \d{4}.*$/, ""),
              releaseDate: createdAt.split("T")[0],
              overview:
                "A compelling story that captivates audiences worldwide with its stunning visuals and masterful storytelling.",
              voteAverage: +(5 + Math.random() * 4).toFixed(1),
              voteCount: randInt(1000, 50000),
            }
          : null,
    });
  }
  return torrents;
}

const ALL_TORRENTS = generateTorrents(50);

function deepMerge(target, source) {
  const BLOCKED_KEYS = new Set(["__proto__", "constructor", "prototype"]);
  for (const key of Object.keys(source)) {
    if (BLOCKED_KEYS.has(key)) continue;
    if (
      Object.prototype.hasOwnProperty.call(target, key) &&
      source[key] !== null && typeof source[key] === "object" && !Array.isArray(source[key]) &&
      target[key] !== null && typeof target[key] === "object" && !Array.isArray(target[key])
    ) {
      deepMerge(target[key], source[key]);
    } else {
      target[key] = source[key];
    }
  }
  return target;
}

function serializeResponse(data) {
  return JSON.stringify({ data });
}

function matchPath(pathname) {
  const downloadMatch = pathname.match(
    /^\/api\/torrents\/([a-f0-9]{40})\/download$/,
  );
  if (downloadMatch) return { route: "download", infoHash: downloadMatch[1] };
  if (pathname === "/graphql") return { route: "graphql" };
  if (pathname === "/api/crawler/status") return { route: "crawlerStatus" };
  return null;
}

function handleGraphQL(body) {
  const opName = body.operationName || extractOpName(body.query);
  const vars = body.variables || {};

  switch (opName) {
    case "HealthCheck":
      return serializeResponse({
        health: {
          status: "up",
          checks: [
            {
              key: "database",
              status: "up",
              timestamp: new Date().toISOString(),
              error: null,
            },
            {
              key: "dht_crawler",
              status: "up",
              timestamp: new Date().toISOString(),
              error: null,
            },
            {
              key: "search",
              status: "up",
              timestamp: new Date().toISOString(),
              error: null,
            },
            {
              key: "queue",
              status: "up",
              timestamp: new Date().toISOString(),
              error: null,
            },
          ],
        },
        version: "0.9.0-mock",
      });

    case "TorrentSearch": {
      const input = vars.input || {};
      const page = input.page || 1;
      const limit = input.limit || 15;
      const contentTypes = input.facets?.contentType?.filter || [];
      let filtered = ALL_TORRENTS;
      if (contentTypes.length > 0) {
        filtered = ALL_TORRENTS.filter((t) =>
          contentTypes.includes(t.contentType),
        );
      }
      if (input.queryString) {
        const q = input.queryString.toLowerCase();
        filtered = filtered.filter(
          (t) => t.title.toLowerCase().includes(q) || t.infoHash.includes(q),
        );
      }
      const totalCount = filtered.length;
      const start = (page - 1) * limit;
      const items = filtered.slice(start, start + limit);
      const hasNextPage = start + limit < totalCount;
      return serializeResponse({
        torrentSearch: {
          search: {
            items: items.map((t) => ({
              infoHash: t.infoHash,
              contentType: t.contentType,
              contentSource: "dht",
              contentId: t.infoHash,
              title: t.title,
              torrent: {
                infoHash: t.infoHash,
                name: t.torrent.name,
                size: t.torrent.size,
                filesCount: t.torrent.filesCount,
                hasFilesInfo: t.torrent.hasFilesInfo,
                fileType: t.torrent.fileType,
                seeders: t.torrent.seeders,
                leechers: t.torrent.leechers,
                magnetUri: t.torrent.magnetUri,
              },
              seeders: t.seeders,
              leechers: t.leechers,
              createdAt: t.createdAt,
            })),
            totalCount,
            totalCountIsEstimate: false,
            hasNextPage,
            barrier: null,
            aggregations: {
              contentType: (() => {
                const counts = {};
                filtered.forEach((t) => {
                  const v = t.contentType;
                  counts[v] = (counts[v] || 0) + 1;
                });
                return Object.entries(counts).map(([value, count]) => ({
                  value,
                  count,
                }));
              })(),
            },
          },
        },
      });
    }

    case "TorrentFiles": {
      const input = vars.input || {};
      const infoHashes = input.infoHashes || [];
      const limit = input.limit || 200;
      let filtered = allFiles.filter((f) => infoHashes.includes(f.infoHash));
      if (!filtered.length && ALL_TORRENTS.length) {
        filtered = allFiles.filter(
          (f) => f.infoHash === ALL_TORRENTS[0].infoHash,
        );
      }
      const items = filtered.slice(0, limit);
      return serializeResponse({
        torrent: {
          files: {
            items: items.map((f) => ({
              infoHash: f.infoHash,
              index: f.index,
              pathParts: f.pathParts,
              size: f.size,
              fileType: f.fileType,
            })),
            totalCount: filtered.length,
          },
        },
      });
    }

    case "TorrentMetrics": {
      const now = Date.now();
      const buckets = [];
      for (let i = 13; i >= 0; i--) {
        const d = new Date(now - i * 86400000);
        buckets.push({
          source: choose(SOURCES),
          bucket: d.toISOString(),
          count: randInt(100, 500),
          updatedCount: randInt(0, 50),
        });
      }
      return serializeResponse({
        torrent: {
          metrics: { buckets },
          listSources: {
            sources: SOURCES.map((k) => ({
              key: k,
              name: "DHT",
            })),
          },
        },
      });
    }

    case "TorrentReprocess":
      return serializeResponse({
        torrent: {
          reprocess: null,
        },
      });

    case "QueueJobs": {
      const input = vars.input || {};
      const limit = input.limit || 15;
      const offset = input.offset || 0;

      const queueNames = [
        "process_torrent",
        "process_torrent_batch",
        "classify_torrent",
        "scrape_metadata",
      ];
      const allJobs = queueNames.map((queue) => ({
        id: `kafka/${queue}`,
        queue,
        payload: JSON.stringify({
          lag: randInt(0, 500),
        }),
        createdAt: new Date(
          Date.now() - randInt(0, 86400) * 1000,
        ).toISOString(),
      }));

      let filtered = allJobs;
      const queueFilter = input.queues;
      if (queueFilter?.values?.length) {
        filtered = filtered.filter((j) => queueFilter.values.includes(j.queue));
      }

      const totalCount = filtered.length;
      const items = filtered.slice(offset, offset + limit);
      const hasNextPage = offset + limit < totalCount;

      const queueAggs = {};
      filtered.forEach((j) => {
        queueAggs[j.queue] = (queueAggs[j.queue] || 0) + 1;
      });

      function aggLabel(v) {
        return v.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
      }

      return serializeResponse({
        queue: {
          jobs: {
            totalCount,
            hasNextPage,
            items: items.map((j) => ({
              id: j.id,
              queue: j.queue,
              payload: j.payload,
              createdAt: j.createdAt,
            })),
            aggregations: {
              queue: Object.entries(queueAggs).map(([value, count]) => ({
                value,
                label: aggLabel(value),
                count,
              })),
            },
          },
        },
      });
    }

    case "Workers":
      return serializeResponse({
        workers: {
          listAll: {
            workers: [
              { key: "dht_crawler", started: true },
              { key: "metadata_scraper", started: true },
              { key: "classifier", started: false },
              { key: "search_indexer", started: true },
            ],
          },
        },
      });

    case "ReindexElasticsearch":
      if (mockReclassify.running && !mockReclassify.done) {
        return serializeResponse({
          torrent: {
            reindexToElasticsearch: { total: 0, indexed: 0, done: true, running: false, error: "another operation in progress: classifier reclassify" },
          },
        });
      }
      mockReindex = { total: 100, indexed: 0, done: false, running: true, error: null };
      return serializeResponse({
        torrent: {
          reindexToElasticsearch: { ...mockReindex },
        },
      });

    case "ReindexStatus":
      if (!mockReindex.done) {
        const progress = Math.floor(Math.random() * 15) + 5;
        mockReindex.indexed = Math.min(mockReindex.total, mockReindex.indexed + progress);
        if (mockReindex.indexed >= mockReindex.total) {
          mockReindex.done = true;
          mockReindex.indexed = mockReindex.total;
        }
      }
      return serializeResponse({
        reindexStatus: { ...mockReindex },
      });

    case "ReclassifyTorrents":
      if (mockReindex.running && !mockReindex.done) {
        return serializeResponse({
          torrent: {
            reclassifyTorrents: { total: 0, processed: 0, done: true, running: false, error: "another operation in progress: embedding reindex" },
          },
        });
      }
      mockReclassify = { total: 100, processed: 0, done: false, running: true, error: null };
      return serializeResponse({
        torrent: {
          reclassifyTorrents: { ...mockReclassify },
        },
      });

    case "ReclassifyStatus":
      if (!mockReclassify.done) {
        const progress = Math.floor(Math.random() * 12) + 4;
        mockReclassify.processed = Math.min(mockReclassify.total, mockReclassify.processed + progress);
        if (mockReclassify.processed >= mockReclassify.total) {
          mockReclassify.done = true;
          mockReclassify.processed = mockReclassify.total;
        }
      }
      return serializeResponse({
        reclassifyStatus: { ...mockReclassify },
      });

    case "Config":
      return serializeResponse({ config: mockConfig });

    case "UpdateConfig":
      if (vars.input) deepMerge(mockConfig, vars.input);
      return serializeResponse({ config: mockConfig });

    case "DhtCrawler": {
      const reindexActive = mockReindex.running && !mockReindex.done;
      const reclassifyActive = mockReclassify.running && !mockReclassify.done;
      const paused = reindexActive || reclassifyActive;
      return serializeResponse({
        dhtCrawler: {
          active: true,
          paused,
          pauseReason: reindexActive ? "embedding reindex" : reclassifyActive ? "classifier reclassify" : null,
          torrentsCrawled: 54321,
          peersConnected: 1234,
          peersDiscovered: 9876,
          uptime: 172800,
          startedAt: new Date(Date.now() - 172800000).toISOString(),
          recentActivity: [
            { id: "1", type: "crawl", message: "Crawled 120 new torrents from DHT", time: new Date().toISOString() },
            { id: "2", type: "peer", message: "Discovered 45 new peers", time: new Date(Date.now() - 60000).toISOString() },
            { id: "3", type: "index", message: "Indexed 80 metadata records", time: new Date(Date.now() - 120000).toISOString() },
          ],
        },
      });
    }

    default:
      return serializeResponse({});
  }
}

function extractOpName(query) {
  if (!query) return null;
  const m = query.match(/(?:query|mutation)\s+(\w+)/);
  return m ? m[1] : null;
}

const server = http.createServer((req, res) => {
  Object.entries(CORS).forEach(([k, v]) => res.setHeader(k, v));

  if (req.method === "OPTIONS") {
    res.writeHead(204);
    res.end();
    return;
  }

  const url = new URL(req.url, `http://${req.headers.host}`);
  const route = matchPath(url.pathname);

  if (!route) {
    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "Not found" }));
    return;
  }

  switch (route.route) {
    case "graphql": {
      if (req.method !== "POST") {
        res.writeHead(405);
        res.end();
        return;
      }
      let body = "";
      req.on("data", (chunk) => (body += chunk));
      req.on("end", () => {
        try {
          const parsed = JSON.parse(body);
          const response = handleGraphQL(parsed);
          res.writeHead(200, { "Content-Type": "application/json" });
          res.end(response);
        } catch {
          res.writeHead(400, { "Content-Type": "application/json" });
          res.end(JSON.stringify({ errors: [{ message: "Invalid JSON" }] }));
        }
      });
      break;
    }

    case "crawlerStatus": {
      const now = Math.floor(Date.now() / 1000);
      const activityTypes = ["discovered", "metadata", "peer", "error"];
      const activityMsgs = {
        discovered: [
          "Torrent 'Interstellar 2014 2160p WEB-DL DD 5 1 H 264' crawled and persisted",
          "Torrent 'The Matrix 1999 2160p BluRay REMUX HEVC' crawled and persisted",
          "Torrent 'Breaking Bad S01E01 1080p WEB-DL' crawled and persisted",
          "Torrent 'Dune Part Two 2024 2160p WEB-DL' crawled and persisted",
          "Torrent hash abcdef0123456789->persisted via DHT announce",
        ],
        metadata: [
          "Metadata resolved for abcdef01 (Interstellar 2014)",
          "Metadata resolved for abcdef02 (The Matrix 1999)",
          "Metadata resolved for abcdef03 (Breaking Bad S01E01)",
        ],
        peer: [
          "Found 12 peers for abcdef01",
          "Found 8 peers for abcdef02",
          "Found 24 peers for abcdef03",
          "Found 6 peers for abcdef04",
          "connecting to 203.0.113.42:6881->abcdef01 via DHT",
          "handshake with 198.51.100.7:6881->abcdef02 completed",
        ],
        error: [
          "Metadata fetch failed for ffffff01: connection timeout",
          "Metadata fetch failed for ffffff02: no peers found",
          "Metadata fetch failed for ffffff03: banned content",
          "read tcp4 172.19.0.5:35440->106.51.110.228:38706: i/o timeout",
          "read tcp4 192.168.1.10:54321->203.0.113.42:6881: connection reset by peer",
          "dial tcp4 10.0.0.1:12345->198.51.100.7:6881: connection refused",
          "write tcp4 172.19.0.5:35441->106.51.110.229:38706: broken pipe",
          "read tcp4 172.19.0.5:35442->106.51.110.230:38706: connection timed out",
        ],
      };
      const recentActivity = [];
      for (let i = 0; i < 30; i++) {
        const type = choose(activityTypes);
        const msgs = activityMsgs[type];
        recentActivity.push({
          id: `act-${String(i).padStart(4, "0")}`,
          type,
          message: choose(msgs),
          time: new Date(Date.now() - randInt(0, 600) * 1000).toISOString(),
        });
      }
      recentActivity.sort(
        (a, b) => new Date(b.time).getTime() - new Date(a.time).getTime(),
      );

      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({
          running: true,
          torrentsCrawled: 1247 + randInt(0, 50),
          torrentsCrawledPerMin: randInt(3, 8),
          peersConnected: 83 + randInt(0, 10),
          peersConnectedPerMin: randInt(0, 2),
          peersDiscovered: 15620 + randInt(0, 100),
          peersDiscoveredPerMin: randInt(20, 50),
          startedAt: now - randInt(3600, 86400),
          recentActivity,
        }),
      );
      break;
    }

    case "download": {
      const dummyTorrent = Buffer.from(
        "d8:announce38:udp://tracker.opentrackr.org:1337/announcee",
      );
      res.writeHead(200, {
        "Content-Type": "application/x-bittorrent",
        "Content-Disposition": 'attachment; filename="mock.torrent"',
        "Content-Length": dummyTorrent.length,
      });
      res.end(dummyTorrent);
      break;
    }
  }
});

server.listen(PORT, () => {
  console.log(`[mock-api] Mock server running on http://localhost:${PORT}`);
  console.log(`[mock-api] GraphQL:  POST http://localhost:${PORT}/graphql`);
  console.log(
    `[mock-api] REST:     GET http://localhost:${PORT}/api/crawler/status`,
  );
});
