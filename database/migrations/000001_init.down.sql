drop table if exists blocked_info_hash cascade;
drop table if exists key_value cascade;
drop table if exists torrent_contents cascade;
drop table if exists content cascade;
drop table if exists torrent_files cascade;
drop table if exists torrents cascade;
drop type if exists "FilesStatus";
drop extension if exists btree_gin;
drop extension if exists pg_trgm;
