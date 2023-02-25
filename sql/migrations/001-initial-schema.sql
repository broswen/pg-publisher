CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
create sequence if not exists version;

create table pg_publisher(
    id text not null primary key ,
    last_published_version bigint not null default 0
);

create table data (
    id uuid default uuid_generate_v4() primary key,
    a text not null default 'test',
    b integer not null default 123,
    c float not null default 123.456,
    d bool not null default true,
    e VARCHAR(16) not null default 'test',
    f TIMESTAMPTZ not null default now(),
    version bigint not null default 0
);

create or replace function increase_version() returns trigger as $$
    begin
       NEW.version := nextval('version');
       return NEW;
    end;
$$ language plpgsql;

create trigger row_modified
    before update or insert
    on data
    for each row
execute procedure increase_version()
