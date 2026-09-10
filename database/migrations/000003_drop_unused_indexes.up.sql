-- Remove indexes that are never used by any query and only consume disk.
-- torrent_files_path_parts_idx: no query filters on path_parts (file search uses torrents.tsv)
-- torrent_files_size_idx: no query orders/filters torrent_files by size
-- torrents_content_type_idx: covered by torrents_content_type_updated_at_idx / torrents_content_type_tsv_idx
-- torrents_content_source_idx: covered by torrents_content_source_content_id_idx
-- content_type_idx: covered by content primary key (type, source, id)
drop index if exists torrent_files_path_parts_idx;
drop index if exists torrent_files_size_idx;
drop index if exists torrents_content_type_idx;
drop index if exists torrents_content_source_idx;
drop index if exists content_type_idx;
