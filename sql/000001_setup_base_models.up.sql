begin;

create table if not exists workspaces (
  id serial primary key,
  created_at timestamptz not null default current_timestamp,
  company_name text not null,
  email_address varchar(50) unique
);

create table if not exists users (
  id serial primary key,
  created_at timestamptz not null default current_timestamp,
  email_address text unique not null,
  password bytea,
  first_name text,
  last_name text,
  role text not null,
  phone_number varchar(20) unique,
  workspace integer not null references workspaces(id)
);

commit;