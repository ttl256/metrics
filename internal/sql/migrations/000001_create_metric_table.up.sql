create table if not exists metric (
    id text primary key,
    type text not null,
    delta bigint,
    value double precision,
    hash text
);
