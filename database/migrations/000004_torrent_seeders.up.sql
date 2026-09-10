-- Move the frequently updated seeder/leecher counters off the wide `torrents`
-- row into a dedicated table. Updating an indexed column on `torrents`
-- (seeders/leechers/updated_at) prevents HOT updates and rewrites the row plus
-- every index, including the large GIN index on torrents.tsv. Keeping the hot
-- counters isolated stops that write amplification and bloat.
create table torrent_seeders
(
  info_hash  text                     not null primary key references torrents on delete cascade,
  seeders    integer,
  leechers   integer,
  updated_at timestamp with time zone not null default now()
);

insert into torrent_seeders (info_hash, seeders, leechers, updated_at)
select info_hash, seeders, leechers, updated_at
from torrents
on conflict (info_hash) do nothing;

create index on torrent_seeders (coalesce(seeders, -1));
create index on torrent_seeders (coalesce(leechers, -1));

drop index if exists torrents_seeders_idx;
drop index if exists torrents_seeders_coalesce_idx;
drop index if exists torrents_leechers_idx;
drop index if exists torrents_leechers_coalesce_idx;

alter table torrents drop column seeders;
alter table torrents drop column leechers;
