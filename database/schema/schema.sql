create extension if not exists pg_trgm;
create extension if not exists btree_gin;

create table content
(
  type              text                     not null,
  source            text                     not null,
  id                text                     not null,
  title             text                     not null,
  release_date      date,
  adult             boolean,
  overview          text,
  popularity        float,
  vote_average      float,
  vote_count        bigint,
  tsv               tsvector                 not null,
  created_at        timestamp with time zone not null,
  updated_at        timestamp with time zone not null,
  primary key (type, source, id)
);

create index on content (type);
create index on content (source);
create index on content (id);
create index on content (release_date);
create index on content (adult);
create index on content (popularity);
create index on content using gin (tsv);

create table torrents
(
  info_hash        text                     not null primary key,
  name             text                     not null,
  size             bigint                   not null,
  private          boolean                  not null,
  files_count      integer,
  content_type     text,
  content_source   text,
  content_id       text,
  languages        jsonb,
  tsv              tsvector                 not null default ''::tsvector,
  seeders          integer,
  leechers         integer,
  created_at       timestamp with time zone not null,
  updated_at       timestamp with time zone not null,
  foreign key (content_type, content_source, content_id) references content (type, source, id) on delete cascade,
  check ((content_type is not null) or (content_id is null)),
  check ((content_source is null) or (content_id is not null))
);

create index on torrents (name);
create index on torrents (size);
create index on torrents (content_type);
create index on torrents (content_source);
create index on torrents (content_id);
create index on torrents (content_source, content_id);
create index on torrents (seeders);
create index on torrents (leechers);
create index on torrents (coalesce(files_count, 0));
create index torrents_seeders_coalesce_idx on torrents (coalesce(seeders, -1));
create index torrents_leechers_coalesce_idx on torrents (coalesce(leechers, -1));
create index on torrents (updated_at);
create index on torrents (content_type, updated_at);
create index on torrents using gin (content_type, tsv);
create index on torrents using gin (content_type, languages);
create index on torrents (created_at);

create table torrent_files
(
  info_hash  text                     not null references torrents on delete cascade,
  "index"    integer                  not null,
  path_parts text[]                   not null default '{}',
  extension  text generated always as
    (substring(lower(path_parts[cardinality(path_parts)]) from '[^.]\.([a-z0-9]+)$')) stored,
  size       bigint                   not null,
  created_at timestamp with time zone not null,
  updated_at timestamp with time zone not null,
  primary key (info_hash, "index")
);

create index on torrent_files (size);
create index on torrent_files (extension);
create index on torrent_files using gin (path_parts);

create table key_value
(
  key        text primary key,
  value      bytea                    not null,
  created_at timestamp with time zone not null,
  updated_at timestamp with time zone not null
);

create table blocked_info_hash
(
  info_hash  text                     not null primary key,
  reason     text                     not null,
  blocked_at timestamp with time zone not null default now()
);

create index on blocked_info_hash (blocked_at);
