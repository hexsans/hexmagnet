alter table torrents add column seeders integer;
alter table torrents add column leechers integer;

update torrents t
set seeders = s.seeders,
    leechers = s.leechers
from torrent_seeders s
where s.info_hash = t.info_hash;

create index on torrents (seeders);
create index on torrents (leechers);
create index torrents_seeders_coalesce_idx on torrents (coalesce(seeders, -1));
create index torrents_leechers_coalesce_idx on torrents (coalesce(leechers, -1));

drop table if exists torrent_seeders;
