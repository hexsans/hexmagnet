create index on torrent_files using gin (path_parts);
create index on torrent_files (size);
create index on torrents (content_type);
create index on torrents (content_source);
create index on content (type);
