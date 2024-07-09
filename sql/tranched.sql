create table if not exists users (
    id serial primary key,
    username text unique not null,
    password bytea not null
);

create table if not exists assets (
    id serial primary key,
    userid int not null,
    asset_type text not null,
    balance float not null
);

create table if not exists orders (
    id serial primary key,
    userid int not null,
    side text not null,
    asset_pair text not null,
    amount float not null,
    price float not null,
    status text not null
);