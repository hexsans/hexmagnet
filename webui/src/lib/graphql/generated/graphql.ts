/* eslint-disable */
/** Internal type. DO NOT USE DIRECTLY. */
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
/** Internal type. DO NOT USE DIRECTLY. */
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
import { TypedDocumentNode as DocumentNode } from '@graphql-typed-document-node/core';
export type ClassifierConfigInput = {
  concurrency?: number | null | undefined;
  llm?: LlmConfigInput | null | undefined;
  tmdb?: TmdbConfigInput | null | undefined;
  torrentFilter?: TorrentFilterConfigInput | null | undefined;
};

export type ConfigInput = {
  classifier?: ClassifierConfigInput | null | undefined;
  dht?: DhtConfigInput | null | undefined;
  server?: ServerConfigInput | null | undefined;
  storage?: StorageConfigInput | null | undefined;
  torznab?: TorznabConfigInput | null | undefined;
  webhooks?: WebhooksConfigInput | null | undefined;
};

export type ContentType =
  | 'adult'
  | 'audiobook'
  | 'comic'
  | 'ebook'
  | 'game'
  | 'movie'
  | 'music'
  | 'other'
  | 'software'
  | 'tv_show'
  | 'unknown';

export type ContentTypeFacetInput = {
  aggregate?: boolean | null | undefined;
  filter?: Array<ContentType | null | undefined> | null | undefined;
};

export type DhtConfigInput = {
  bootstrapNodes?: Array<string> | null | undefined;
  port?: number | null | undefined;
  requester?: DhtRequesterInput | null | undefined;
  reseedBootstrapNodesInterval?: number | null | undefined;
  responder?: DhtResponderConfigInput | null | undefined;
};

export type DhtRequesterInput = {
  hashDiscoverLimit?: number | null | undefined;
  requestLimit?: number | null | undefined;
  rescrapeThreshold?: number | null | undefined;
};

export type DhtResponderConfigInput = {
  enabled?: boolean | null | undefined;
  globalRateLimit?: number | null | undefined;
  perIPRateLimit?: number | null | undefined;
};

export type ElasticsearchConfigInput = {
  addresses?: Array<string> | null | undefined;
  embedding?: EmbeddingConfigInput | null | undefined;
};

export type EmbeddingConfigInput = {
  apikey?: string | null | undefined;
  dimensions?: number | null | undefined;
  endpoint?: string | null | undefined;
  instructionEnabled?: boolean | null | undefined;
  model?: string | null | undefined;
};

export type FacetLogic =
  | 'and'
  | 'or';

export type FileType =
  | 'archive'
  | 'audio'
  | 'data'
  | 'document'
  | 'image'
  | 'software'
  | 'subtitles'
  | 'video';

export type HealthStatus =
  | 'down'
  | 'inactive'
  | 'unknown'
  | 'up';

export type KafkaConfigInput = {
  brokers?: Array<string> | null | undefined;
};

export type LlmConfigInput = {
  apiKey?: string | null | undefined;
  enabled?: boolean | null | undefined;
  endpoint?: string | null | undefined;
  maxFiles?: number | null | undefined;
  maxRetries?: number | null | undefined;
  model?: string | null | undefined;
  reasoningEffort?: string | null | undefined;
  temperature?: number | null | undefined;
  timeout?: number | null | undefined;
};

export type Language =
  | 'af'
  | 'ar'
  | 'az'
  | 'be'
  | 'bg'
  | 'bs'
  | 'ca'
  | 'ce'
  | 'co'
  | 'cs'
  | 'cy'
  | 'da'
  | 'de'
  | 'el'
  | 'en'
  | 'es'
  | 'et'
  | 'eu'
  | 'fa'
  | 'fi'
  | 'fr'
  | 'he'
  | 'hi'
  | 'hr'
  | 'hu'
  | 'hy'
  | 'id'
  | 'is'
  | 'it'
  | 'ja'
  | 'ka'
  | 'ko'
  | 'ku'
  | 'lt'
  | 'lv'
  | 'mi'
  | 'mk'
  | 'ml'
  | 'mn'
  | 'ms'
  | 'mt'
  | 'nl'
  | 'no'
  | 'pl'
  | 'pt'
  | 'ro'
  | 'ru'
  | 'sa'
  | 'sk'
  | 'sl'
  | 'sm'
  | 'so'
  | 'sr'
  | 'sv'
  | 'ta'
  | 'th'
  | 'tr'
  | 'uk'
  | 'vi'
  | 'yi'
  | 'zh'
  | 'zu';

export type LanguageFacetInput = {
  aggregate?: boolean | null | undefined;
  filter?: Array<Language> | null | undefined;
};

export type MetricsBucketDuration =
  | 'day'
  | 'hour'
  | 'minute';

export type PostgresConfigInput = {
  connectionTimeout?: number | null | undefined;
  database?: string | null | undefined;
  host?: string | null | undefined;
  maxConnections?: number | null | undefined;
  password?: string | null | undefined;
  port?: number | null | undefined;
  sslCertPath?: string | null | undefined;
  sslKeyPath?: string | null | undefined;
  sslMode?: string | null | undefined;
  sslRootCertPath?: string | null | undefined;
  username?: string | null | undefined;
};

export type QueueConfigInput = {
  backend?: string | null | undefined;
  kafka?: KafkaConfigInput | null | undefined;
};

export type QueueJobQueueFacetInput = {
  logic: FacetLogic;
  values: Array<string>;
};

export type QueueJobsOrderByField =
  | 'createdAt'
  | 'priority'
  | 'ranAt';

export type QueueJobsOrderByInput = {
  descending?: boolean | null | undefined;
  field: QueueJobsOrderByField;
};

export type QueueJobsQueryInput = {
  limit?: number | null | undefined;
  offset?: number | null | undefined;
  orderBy?: QueueJobsOrderByInput | null | undefined;
  queues?: QueueJobQueueFacetInput | null | undefined;
};

export type ReleaseYearFacetInput = {
  aggregate?: boolean | null | undefined;
  filter?: Array<number | null | undefined> | null | undefined;
};

export type SearchConfigInput = {
  backend?: string | null | undefined;
  elasticsearch?: ElasticsearchConfigInput | null | undefined;
};

export type ServerConfigInput = {
  embedTrackers?: Array<string> | null | undefined;
  ip?: string | null | undefined;
  log?: ServerLogConfigInput | null | undefined;
  port?: number | null | undefined;
  torrentFilePath?: string | null | undefined;
};

export type ServerFileRotatorConfigInput = {
  format?: string | null | undefined;
  maxBackups?: number | null | undefined;
  path?: string | null | undefined;
};

export type ServerLogConfigInput = {
  consoleLevel?: string | null | undefined;
  fileOutputLevel?: string | null | undefined;
  fileRotator?: ServerFileRotatorConfigInput | null | undefined;
};

export type SortDirection =
  | 'asc'
  | 'desc';

export type StorageConfigInput = {
  postgres?: PostgresConfigInput | null | undefined;
  queue?: QueueConfigInput | null | undefined;
  search?: SearchConfigInput | null | undefined;
};

export type TmdbConfigInput = {
  accessToken?: string | null | undefined;
  enabled?: boolean | null | undefined;
  rateLimit?: number | null | undefined;
};

export type TorrentFileTypeFacetInput = {
  aggregate?: boolean | null | undefined;
  filter?: Array<FileType> | null | undefined;
  logic?: FacetLogic | null | undefined;
};

export type TorrentFilesOrderByField =
  | 'extension'
  | 'index'
  | 'size';

export type TorrentFilesOrderByInput = {
  descending?: boolean | null | undefined;
  field: TorrentFilesOrderByField;
};

export type TorrentFilesQueryInput = {
  cached?: boolean | null | undefined;
  hasNextPage?: boolean | null | undefined;
  infoHashes?: Array<string> | null | undefined;
  limit?: number | null | undefined;
  offset?: number | null | undefined;
  orderBy?: Array<TorrentFilesOrderByInput> | null | undefined;
  page?: number | null | undefined;
  totalCount?: boolean | null | undefined;
};

export type TorrentFilterConfigInput = {
  filenamePatterns?: Array<string> | null | undefined;
  mode?: string | null | undefined;
  titlePatterns?: Array<string> | null | undefined;
};

export type TorrentMetricsQueryInput = {
  bucketDuration: MetricsBucketDuration;
  endTime?: string | null | undefined;
  startTime?: string | null | undefined;
  timezone?: string | null | undefined;
};

export type TorrentReprocessInput = {
  classifierRematch?: boolean | null | undefined;
  contentType?: ContentType | null | undefined;
  infoHashes: Array<string>;
};

export type TorrentSearchFacetsInput = {
  contentType?: ContentTypeFacetInput | null | undefined;
  language?: LanguageFacetInput | null | undefined;
  releaseYear?: ReleaseYearFacetInput | null | undefined;
  torrentFileType?: TorrentFileTypeFacetInput | null | undefined;
};

export type TorrentSearchOrderByField =
  | 'created_at'
  | 'files_count'
  | 'info_hash'
  | 'leechers'
  | 'name'
  | 'relevance'
  | 'seeders'
  | 'size'
  | 'updated_at';

export type TorrentSearchOrderByInput = {
  direction?: SortDirection;
  field: TorrentSearchOrderByField;
};

export type TorrentSearchQueryInput = {
  aggregationBudget?: number | null | undefined;
  /**
   * ISO 8601 timestamp. When set, only content created at or before this timestamp
   * is included — freezing the result set against newly indexed content during pagination.
   */
  barrier?: string | null | undefined;
  cached?: boolean | null | undefined;
  facets?: TorrentSearchFacetsInput | null | undefined;
  /** hasNextPage if true, the search result will include the hasNextPage field, indicating if there are more results to fetch */
  hasNextPage?: boolean | null | undefined;
  infoHashes?: Array<string> | null | undefined;
  limit?: number | null | undefined;
  offset?: number | null | undefined;
  orderBy?: Array<TorrentSearchOrderByInput> | null | undefined;
  page?: number | null | undefined;
  queryString?: string | null | undefined;
  totalCount?: boolean | null | undefined;
};

export type TorznabConfigInput = {
  apiKey?: string | null | undefined;
  categories?: Array<string> | null | undefined;
  enabled?: boolean | null | undefined;
  maxResults?: number | null | undefined;
  path?: string | null | undefined;
  trustProxyHeaders?: boolean | null | undefined;
};

export type WebhookHeaderInput = {
  key: string;
  value: string;
};

export type WebhooksConfigInput = {
  baseUrl?: string | null | undefined;
  categories?: Array<string> | null | undefined;
  enabled?: boolean | null | undefined;
  events?: Array<string> | null | undefined;
  filenamePatterns?: Array<string> | null | undefined;
  headers?: Array<WebhookHeaderInput> | null | undefined;
  maxRetries?: number | null | undefined;
  queueSize?: number | null | undefined;
  timeout?: number | null | undefined;
  titlePatterns?: Array<string> | null | undefined;
  urls?: Array<string> | null | undefined;
};

export type ReclassifyTorrentsMutationVariables = Exact<{ [key: string]: never; }>;


export type ReclassifyTorrentsMutation = { torrent: { reclassifyTorrents: { total: number, processed: number, done: boolean, running: boolean, error: string | null } } };

export type ReindexElasticsearchMutationVariables = Exact<{ [key: string]: never; }>;


export type ReindexElasticsearchMutation = { torrent: { reindexToElasticsearch: { total: number, indexed: number, done: boolean, error: string | null } } };

export type TorrentReprocessMutationVariables = Exact<{
  input: TorrentReprocessInput;
}>;


export type TorrentReprocessMutation = { torrent: { reprocess: void | null } };

export type UpdateConfigMutationVariables = Exact<{
  input: ConfigInput;
}>;


export type UpdateConfigMutation = { updateConfig: { dht: { port: number, bootstrapNodes: Array<string>, reseedBootstrapNodesInterval: number, responder: { enabled: boolean, globalRateLimit: number, perIPRateLimit: number }, requester: { requestLimit: number, rescrapeThreshold: number, hashDiscoverLimit: number } }, server: { ip: string, port: number, embedTrackers: Array<string>, log: { consoleLevel: string, fileOutputLevel: string, fileRotator: { path: string, maxBackups: number, format: string } } }, classifier: { concurrency: number, llm: { endpoint: string, apiKey: string, model: string, timeout: number, maxRetries: number, temperature: number, reasoningEffort: string, maxFiles: number, enabled: boolean }, torrentFilter: { mode: string, titlePatterns: Array<string>, filenamePatterns: Array<string> }, tmdb: { enabled: boolean, accessToken: string, rateLimit: number } }, storage: { postgres: { host: string, username: string, port: number, database: string, password: string, sslMode: string, connectionTimeout: number, sslCertPath: string, sslKeyPath: string, sslRootCertPath: string, maxConnections: number }, search: { backend: string, elasticsearch: { addresses: Array<string>, embedding: { endpoint: string, apikey: string, model: string, dimensions: number, instructionEnabled: boolean } } }, queue: { backend: string, kafka: { brokers: Array<string> } } }, torznab: { enabled: boolean, apiKey: string, path: string, maxResults: number, categories: Array<string>, trustProxyHeaders: boolean }, webhooks: { enabled: boolean, urls: Array<string>, events: Array<string>, categories: Array<string>, titlePatterns: Array<string>, filenamePatterns: Array<string>, timeout: number, maxRetries: number, baseUrl: string, queueSize: number, headers: Array<{ key: string, value: string }> } } };

export type ConfigQueryVariables = Exact<{ [key: string]: never; }>;


export type ConfigQuery = { config: { dht: { port: number, bootstrapNodes: Array<string>, reseedBootstrapNodesInterval: number, responder: { enabled: boolean, globalRateLimit: number, perIPRateLimit: number }, requester: { requestLimit: number, rescrapeThreshold: number, hashDiscoverLimit: number } }, server: { ip: string, port: number, embedTrackers: Array<string>, torrentFilePath: string, log: { consoleLevel: string, fileOutputLevel: string, fileRotator: { path: string, maxBackups: number, format: string } } }, classifier: { concurrency: number, llm: { endpoint: string, apiKey: string, model: string, timeout: number, maxRetries: number, temperature: number, reasoningEffort: string, maxFiles: number, enabled: boolean }, torrentFilter: { mode: string, titlePatterns: Array<string>, filenamePatterns: Array<string> }, tmdb: { enabled: boolean, accessToken: string, rateLimit: number } }, storage: { postgres: { host: string, username: string, port: number, database: string, password: string, sslMode: string, connectionTimeout: number, sslCertPath: string, sslKeyPath: string, sslRootCertPath: string, maxConnections: number }, search: { backend: string, elasticsearch: { addresses: Array<string>, embedding: { endpoint: string, apikey: string, model: string, dimensions: number, instructionEnabled: boolean } } }, queue: { backend: string, kafka: { brokers: Array<string> } } }, torznab: { enabled: boolean, apiKey: string, path: string, maxResults: number, categories: Array<string>, trustProxyHeaders: boolean }, webhooks: { enabled: boolean, urls: Array<string>, events: Array<string>, categories: Array<string>, titlePatterns: Array<string>, filenamePatterns: Array<string>, timeout: number, maxRetries: number, baseUrl: string, queueSize: number, headers: Array<{ key: string, value: string }> } } };

export type DhtCrawlerQueryVariables = Exact<{ [key: string]: never; }>;


export type DhtCrawlerQuery = { dhtCrawler: { active: boolean, paused: boolean, pauseReason: string | null, torrentsCrawled: number, peersConnected: number, peersDiscovered: number, uptime: number, startedAt: string | null, recentActivity: Array<{ id: string, type: string, message: string, time: string }> } };

export type HealthCheckQueryVariables = Exact<{ [key: string]: never; }>;


export type HealthCheckQuery = { version: string, health: { status: HealthStatus, checks: Array<{ key: string, status: HealthStatus, timestamp: string, error: string | null }> } };

export type QueueJobsQueryVariables = Exact<{
  input: QueueJobsQueryInput;
}>;


export type QueueJobsQuery = { queue: { jobs: { totalCount: number, hasNextPage: boolean | null, items: Array<{ id: string, queue: string, payload: string }>, aggregations: { queue: Array<{ value: string, label: string, count: number }> | null } } } };

export type ReclassifyStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type ReclassifyStatusQuery = { reclassifyStatus: { total: number, processed: number, done: boolean, running: boolean, error: string | null } };

export type ReindexStatusQueryVariables = Exact<{ [key: string]: never; }>;


export type ReindexStatusQuery = { reindexStatus: { total: number, indexed: number, done: boolean, running: boolean, error: string | null } };

export type TorrentFilesQueryVariables = Exact<{
  input: TorrentFilesQueryInput;
}>;


export type TorrentFilesQuery = { torrent: { files: { totalCount: number, items: Array<{ infoHash: string, index: number, pathParts: Array<string>, size: number, fileType: FileType | null }> } } };

export type TorrentMetricsQueryVariables = Exact<{
  input: TorrentMetricsQueryInput;
}>;


export type TorrentMetricsQuery = { torrent: { metrics: { buckets: Array<{ bucket: string, count: number, updatedCount: number }> } } };

export type TorrentSearchQueryVariables = Exact<{
  input: TorrentSearchQueryInput;
}>;


export type TorrentSearchQuery = { torrentSearch: { search: { totalCount: number, totalCountIsEstimate: boolean, hasNextPage: boolean | null, barrier: string | null, items: Array<{ infoHash: string, contentType: ContentType | null, contentSource: string | null, contentId: string | null, title: string, seeders: number | null, leechers: number | null, createdAt: string, torrent: { infoHash: string, name: string, size: number, filesCount: number | null, hasFilesInfo: boolean, fileType: FileType | null, seeders: number | null, leechers: number | null, magnetUri: string } }>, aggregations: { contentType: Array<{ value: ContentType | null, count: number }> | null } } } };

export type WorkersQueryVariables = Exact<{ [key: string]: never; }>;


export type WorkersQuery = { workers: { listAll: { workers: Array<{ key: string, started: boolean }> } } };


export const ReclassifyTorrentsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ReclassifyTorrents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reclassifyTorrents"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"processed"}},{"kind":"Field","name":{"kind":"Name","value":"done"}},{"kind":"Field","name":{"kind":"Name","value":"running"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]}}]} as unknown as DocumentNode<ReclassifyTorrentsMutation, ReclassifyTorrentsMutationVariables>;
export const ReindexElasticsearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"ReindexElasticsearch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reindexToElasticsearch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"indexed"}},{"kind":"Field","name":{"kind":"Name","value":"done"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]}}]} as unknown as DocumentNode<ReindexElasticsearchMutation, ReindexElasticsearchMutationVariables>;
export const TorrentReprocessDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"TorrentReprocess"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentReprocessInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reprocess"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}]}]}}]}}]} as unknown as DocumentNode<TorrentReprocessMutation, TorrentReprocessMutationVariables>;
export const UpdateConfigDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"mutation","name":{"kind":"Name","value":"UpdateConfig"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"ConfigInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"updateConfig"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dht"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"port"}},{"kind":"Field","name":{"kind":"Name","value":"responder"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"globalRateLimit"}},{"kind":"Field","name":{"kind":"Name","value":"perIPRateLimit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"requester"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestLimit"}},{"kind":"Field","name":{"kind":"Name","value":"rescrapeThreshold"}},{"kind":"Field","name":{"kind":"Name","value":"hashDiscoverLimit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"bootstrapNodes"}},{"kind":"Field","name":{"kind":"Name","value":"reseedBootstrapNodesInterval"}}]}},{"kind":"Field","name":{"kind":"Name","value":"server"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ip"}},{"kind":"Field","name":{"kind":"Name","value":"port"}},{"kind":"Field","name":{"kind":"Name","value":"embedTrackers"}},{"kind":"Field","name":{"kind":"Name","value":"log"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"consoleLevel"}},{"kind":"Field","name":{"kind":"Name","value":"fileOutputLevel"}},{"kind":"Field","name":{"kind":"Name","value":"fileRotator"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"maxBackups"}},{"kind":"Field","name":{"kind":"Name","value":"format"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"classifier"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"}},{"kind":"Field","name":{"kind":"Name","value":"llm"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"model"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"maxRetries"}},{"kind":"Field","name":{"kind":"Name","value":"temperature"}},{"kind":"Field","name":{"kind":"Name","value":"reasoningEffort"}},{"kind":"Field","name":{"kind":"Name","value":"maxFiles"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFilter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"titlePatterns"}},{"kind":"Field","name":{"kind":"Name","value":"filenamePatterns"}}]}},{"kind":"Field","name":{"kind":"Name","value":"tmdb"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"accessToken"}},{"kind":"Field","name":{"kind":"Name","value":"rateLimit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"storage"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"postgres"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"host"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"port"}},{"kind":"Field","name":{"kind":"Name","value":"database"}},{"kind":"Field","name":{"kind":"Name","value":"password"}},{"kind":"Field","name":{"kind":"Name","value":"sslMode"}},{"kind":"Field","name":{"kind":"Name","value":"connectionTimeout"}},{"kind":"Field","name":{"kind":"Name","value":"sslCertPath"}},{"kind":"Field","name":{"kind":"Name","value":"sslKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"sslRootCertPath"}},{"kind":"Field","name":{"kind":"Name","value":"maxConnections"}}]}},{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"backend"}},{"kind":"Field","name":{"kind":"Name","value":"elasticsearch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addresses"}},{"kind":"Field","name":{"kind":"Name","value":"embedding"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apikey"}},{"kind":"Field","name":{"kind":"Name","value":"model"}},{"kind":"Field","name":{"kind":"Name","value":"dimensions"}},{"kind":"Field","name":{"kind":"Name","value":"instructionEnabled"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"queue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"backend"}},{"kind":"Field","name":{"kind":"Name","value":"kafka"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brokers"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torznab"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"maxResults"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}},{"kind":"Field","name":{"kind":"Name","value":"trustProxyHeaders"}}]}},{"kind":"Field","name":{"kind":"Name","value":"webhooks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"urls"}},{"kind":"Field","name":{"kind":"Name","value":"events"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}},{"kind":"Field","name":{"kind":"Name","value":"titlePatterns"}},{"kind":"Field","name":{"kind":"Name","value":"filenamePatterns"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"maxRetries"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"headers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"queueSize"}}]}}]}}]}}]} as unknown as DocumentNode<UpdateConfigMutation, UpdateConfigMutationVariables>;
export const ConfigDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Config"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"config"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dht"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"port"}},{"kind":"Field","name":{"kind":"Name","value":"responder"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"globalRateLimit"}},{"kind":"Field","name":{"kind":"Name","value":"perIPRateLimit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"requester"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"requestLimit"}},{"kind":"Field","name":{"kind":"Name","value":"rescrapeThreshold"}},{"kind":"Field","name":{"kind":"Name","value":"hashDiscoverLimit"}}]}},{"kind":"Field","name":{"kind":"Name","value":"bootstrapNodes"}},{"kind":"Field","name":{"kind":"Name","value":"reseedBootstrapNodesInterval"}}]}},{"kind":"Field","name":{"kind":"Name","value":"server"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"ip"}},{"kind":"Field","name":{"kind":"Name","value":"port"}},{"kind":"Field","name":{"kind":"Name","value":"embedTrackers"}},{"kind":"Field","name":{"kind":"Name","value":"torrentFilePath"}},{"kind":"Field","name":{"kind":"Name","value":"log"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"consoleLevel"}},{"kind":"Field","name":{"kind":"Name","value":"fileOutputLevel"}},{"kind":"Field","name":{"kind":"Name","value":"fileRotator"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"maxBackups"}},{"kind":"Field","name":{"kind":"Name","value":"format"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"classifier"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"concurrency"}},{"kind":"Field","name":{"kind":"Name","value":"llm"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"model"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"maxRetries"}},{"kind":"Field","name":{"kind":"Name","value":"temperature"}},{"kind":"Field","name":{"kind":"Name","value":"reasoningEffort"}},{"kind":"Field","name":{"kind":"Name","value":"maxFiles"}},{"kind":"Field","name":{"kind":"Name","value":"enabled"}}]}},{"kind":"Field","name":{"kind":"Name","value":"torrentFilter"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"mode"}},{"kind":"Field","name":{"kind":"Name","value":"titlePatterns"}},{"kind":"Field","name":{"kind":"Name","value":"filenamePatterns"}}]}},{"kind":"Field","name":{"kind":"Name","value":"tmdb"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"accessToken"}},{"kind":"Field","name":{"kind":"Name","value":"rateLimit"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"storage"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"postgres"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"host"}},{"kind":"Field","name":{"kind":"Name","value":"username"}},{"kind":"Field","name":{"kind":"Name","value":"port"}},{"kind":"Field","name":{"kind":"Name","value":"database"}},{"kind":"Field","name":{"kind":"Name","value":"password"}},{"kind":"Field","name":{"kind":"Name","value":"sslMode"}},{"kind":"Field","name":{"kind":"Name","value":"connectionTimeout"}},{"kind":"Field","name":{"kind":"Name","value":"sslCertPath"}},{"kind":"Field","name":{"kind":"Name","value":"sslKeyPath"}},{"kind":"Field","name":{"kind":"Name","value":"sslRootCertPath"}},{"kind":"Field","name":{"kind":"Name","value":"maxConnections"}}]}},{"kind":"Field","name":{"kind":"Name","value":"search"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"backend"}},{"kind":"Field","name":{"kind":"Name","value":"elasticsearch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"addresses"}},{"kind":"Field","name":{"kind":"Name","value":"embedding"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"endpoint"}},{"kind":"Field","name":{"kind":"Name","value":"apikey"}},{"kind":"Field","name":{"kind":"Name","value":"model"}},{"kind":"Field","name":{"kind":"Name","value":"dimensions"}},{"kind":"Field","name":{"kind":"Name","value":"instructionEnabled"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"queue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"backend"}},{"kind":"Field","name":{"kind":"Name","value":"kafka"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"brokers"}}]}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"torznab"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"apiKey"}},{"kind":"Field","name":{"kind":"Name","value":"path"}},{"kind":"Field","name":{"kind":"Name","value":"maxResults"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}},{"kind":"Field","name":{"kind":"Name","value":"trustProxyHeaders"}}]}},{"kind":"Field","name":{"kind":"Name","value":"webhooks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"enabled"}},{"kind":"Field","name":{"kind":"Name","value":"urls"}},{"kind":"Field","name":{"kind":"Name","value":"events"}},{"kind":"Field","name":{"kind":"Name","value":"categories"}},{"kind":"Field","name":{"kind":"Name","value":"titlePatterns"}},{"kind":"Field","name":{"kind":"Name","value":"filenamePatterns"}},{"kind":"Field","name":{"kind":"Name","value":"timeout"}},{"kind":"Field","name":{"kind":"Name","value":"maxRetries"}},{"kind":"Field","name":{"kind":"Name","value":"baseUrl"}},{"kind":"Field","name":{"kind":"Name","value":"headers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"value"}}]}},{"kind":"Field","name":{"kind":"Name","value":"queueSize"}}]}}]}}]}}]} as unknown as DocumentNode<ConfigQuery, ConfigQueryVariables>;
export const DhtCrawlerDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"DhtCrawler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"dhtCrawler"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"active"}},{"kind":"Field","name":{"kind":"Name","value":"paused"}},{"kind":"Field","name":{"kind":"Name","value":"pauseReason"}},{"kind":"Field","name":{"kind":"Name","value":"torrentsCrawled"}},{"kind":"Field","name":{"kind":"Name","value":"peersConnected"}},{"kind":"Field","name":{"kind":"Name","value":"peersDiscovered"}},{"kind":"Field","name":{"kind":"Name","value":"uptime"}},{"kind":"Field","name":{"kind":"Name","value":"startedAt"}},{"kind":"Field","name":{"kind":"Name","value":"recentActivity"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"type"}},{"kind":"Field","name":{"kind":"Name","value":"message"}},{"kind":"Field","name":{"kind":"Name","value":"time"}}]}}]}}]}}]} as unknown as DocumentNode<DhtCrawlerQuery, DhtCrawlerQueryVariables>;
export const HealthCheckDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"HealthCheck"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"health"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"checks"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"status"}},{"kind":"Field","name":{"kind":"Name","value":"timestamp"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"version"}}]}}]} as unknown as DocumentNode<HealthCheckQuery, HealthCheckQueryVariables>;
export const QueueJobsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"QueueJobs"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"QueueJobsQueryInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"jobs"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"id"}},{"kind":"Field","name":{"kind":"Name","value":"queue"}},{"kind":"Field","name":{"kind":"Name","value":"payload"}}]}},{"kind":"Field","name":{"kind":"Name","value":"aggregations"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"queue"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"label"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}}]}}]}}]}}]} as unknown as DocumentNode<QueueJobsQuery, QueueJobsQueryVariables>;
export const ReclassifyStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ReclassifyStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reclassifyStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"processed"}},{"kind":"Field","name":{"kind":"Name","value":"done"}},{"kind":"Field","name":{"kind":"Name","value":"running"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<ReclassifyStatusQuery, ReclassifyStatusQueryVariables>;
export const ReindexStatusDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"ReindexStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"reindexStatus"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"total"}},{"kind":"Field","name":{"kind":"Name","value":"indexed"}},{"kind":"Field","name":{"kind":"Name","value":"done"}},{"kind":"Field","name":{"kind":"Name","value":"running"}},{"kind":"Field","name":{"kind":"Name","value":"error"}}]}}]}}]} as unknown as DocumentNode<ReindexStatusQuery, ReindexStatusQueryVariables>;
export const TorrentFilesDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TorrentFiles"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentFilesQueryInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"files"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"index"}},{"kind":"Field","name":{"kind":"Name","value":"pathParts"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"fileType"}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}}]}}]}}]}}]} as unknown as DocumentNode<TorrentFilesQuery, TorrentFilesQueryVariables>;
export const TorrentMetricsDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TorrentMetrics"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentMetricsQueryInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"metrics"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"buckets"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"bucket"}},{"kind":"Field","name":{"kind":"Name","value":"count"}},{"kind":"Field","name":{"kind":"Name","value":"updatedCount"}}]}}]}}]}}]}}]} as unknown as DocumentNode<TorrentMetricsQuery, TorrentMetricsQueryVariables>;
export const TorrentSearchDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"TorrentSearch"},"variableDefinitions":[{"kind":"VariableDefinition","variable":{"kind":"Variable","name":{"kind":"Name","value":"input"}},"type":{"kind":"NonNullType","type":{"kind":"NamedType","name":{"kind":"Name","value":"TorrentSearchQueryInput"}}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"torrentSearch"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"search"},"arguments":[{"kind":"Argument","name":{"kind":"Name","value":"input"},"value":{"kind":"Variable","name":{"kind":"Name","value":"input"}}}],"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"items"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"contentType"}},{"kind":"Field","name":{"kind":"Name","value":"contentSource"}},{"kind":"Field","name":{"kind":"Name","value":"contentId"}},{"kind":"Field","name":{"kind":"Name","value":"title"}},{"kind":"Field","name":{"kind":"Name","value":"torrent"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"infoHash"}},{"kind":"Field","name":{"kind":"Name","value":"name"}},{"kind":"Field","name":{"kind":"Name","value":"size"}},{"kind":"Field","name":{"kind":"Name","value":"filesCount"}},{"kind":"Field","name":{"kind":"Name","value":"hasFilesInfo"}},{"kind":"Field","name":{"kind":"Name","value":"fileType"}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"leechers"}},{"kind":"Field","name":{"kind":"Name","value":"magnetUri"}}]}},{"kind":"Field","name":{"kind":"Name","value":"seeders"}},{"kind":"Field","name":{"kind":"Name","value":"leechers"}},{"kind":"Field","name":{"kind":"Name","value":"createdAt"}}]}},{"kind":"Field","name":{"kind":"Name","value":"aggregations"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"contentType"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"value"}},{"kind":"Field","name":{"kind":"Name","value":"count"}}]}}]}},{"kind":"Field","name":{"kind":"Name","value":"totalCount"}},{"kind":"Field","name":{"kind":"Name","value":"totalCountIsEstimate"}},{"kind":"Field","name":{"kind":"Name","value":"hasNextPage"}},{"kind":"Field","name":{"kind":"Name","value":"barrier"}}]}}]}}]}}]} as unknown as DocumentNode<TorrentSearchQuery, TorrentSearchQueryVariables>;
export const WorkersDocument = {"kind":"Document","definitions":[{"kind":"OperationDefinition","operation":"query","name":{"kind":"Name","value":"Workers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"listAll"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"workers"},"selectionSet":{"kind":"SelectionSet","selections":[{"kind":"Field","name":{"kind":"Name","value":"key"}},{"kind":"Field","name":{"kind":"Name","value":"started"}}]}}]}}]}}]}}]} as unknown as DocumentNode<WorkersQuery, WorkersQueryVariables>;