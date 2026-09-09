create table torrent_retry_queue
(
  info_hash       text                     not null,
  stage           text                     not null,
  payload         jsonb                    not null,
  fail_count      integer                  not null default 1,
  last_error      text                     not null default '',
  last_failure_at timestamp with time zone not null,
  next_retry_at   timestamp with time zone not null,
  dispatched_at   timestamp with time zone,
  primary key (info_hash, stage)
);

create index on torrent_retry_queue (next_retry_at);
create index on torrent_retry_queue (stage);
