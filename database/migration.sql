-- DB Migration Script

-- To Reset all the events and tasks, and clean out all the schedules
TRUNCATE event RESTART IDENTITY;
TRUNCATE task RESTART IDENTITY;
TRUNCATE task_check RESTART IDENTITY;
TRUNCATE task_part RESTART IDENTITY;
TRUNCATE sched_task RESTART IDENTITY;
TRUNCATE sched_task_part RESTART IDENTITY;


-- 2016-05-11  
-- Modify task part records

alter table task add labour_hrs numeric(12,2) not null default 0;
alter table task_part add qty_used numeric(12,2) not null default 0;
alter table part_stock add descr text not null default '';
alter table part_price add descr text not null default '';

-- Capture SMS transmissions

drop table if exists sms_trans;
create table sms_trans (
	id serial not null primary key,
	number_to text not null default '',
	number_used text not null default '',
	user_id int not null default 0,
	message text not null default '',
	date_sent timestamptz not null default localtimestamp,
	ref text not null default '',
	status text not null default '',
	error text not null default ''
);

-- 2016-05-12
-- Modify user to have hourly rate, and seq task IDs by site

alter table users add hourly_rate numeric(12,2) not null default 0;
alter table users add address text not null default '';
alter table users add site_id int not null default 0;
alter table users add notes text not null default '';

-- 2016-05-16
-- Syslog has a more useful fields

alter table user_log 
add channel int not null default 0,
add user_id int not null default 0,
add entity text not null default '',
add entity_id int not null default 0,
add error text not null default '',
add is_update bool not null default false;

-- Parts tree
alter table part add category int not null default 0;
create table category (
	id serial not null primary key,
	parent_id int not null default 0,
	name text not null default '',
	descr text not null default ''
);

create table site_category (
	site_id int not null,
	cat_id int not null
);
create index site_category_idx on site_category (site_id, cat_id);

-- 2016-06-03 
-- Fix up machine layout for Chinderrah and Connecticut
delete from site_layout where site_id=8;
delete from site_layout where site_id=9;

insert into site_layout (site_id, seq, machine_id, span) values
(8,1,26,12),
(8,2,22,12),
(8,3,23,12),
(8,4,25,12),
(8,5,24,12),
(9,1,40,12),
(9,2,41,12),
(9,3,39,12),
(9,4,38,12),
(9,5,37,12),
(9,6,42,12),
(9,7,43,12);

-- 2016-06-13
-- MachineTypes database

drop table if exists machine_type;
create table machine_type (
	id serial not null primary key,
	name text not null default '',
	electrical bool default true,
	hydraulic bool default true,
	pnuematic bool default true,
	lube bool default true,
	printer bool default true,
	console bool default true,
	uncoiler bool default true,
	rollbed bool  default true
);

insert into machine_type (name) 
values ('Bracket'),('Stud'),('Chord'),('Plate'),('Web'),('Floor'),('Valley'),('Top Hat 22'),('Top Hat 40');

alter table machine add pnuematic text not null default 'Running';

drop table if exists machine_type_tool;
create table machine_type_tool (
	machine_id int not null,
	position int not null default 0,
	name text not null default ''
);

create index machine_type_tool_idx on machine_type_tool (machine_id, position);

insert into machine_type_tool (machine_id, position, name)
values (1,1,'Guillo'),
(2,1,'Brick Tie'),(2,2,'Service Hole #1'),(2,3,'Quad Dimple'),(2,4,'Service Hole #2'),(2,5,'Single Dimple & Rib'),(2,6,'Curl #1'),(2,7,'Guillo'),(2,8,'Curl #2'),
(3,1,'Down Dimple'),
(4,1,'Single Dimple Square');
