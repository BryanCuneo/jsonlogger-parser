if  exists (select * from sys.objects where object_id = object_id('[dbo].[log_entries]') and type in ('u'))
    drop table [dbo].[log_entries]

if  exists (select * from sys.objects where object_id = object_id('[dbo].[log_sessions]') and type in ('u'))
    drop table [dbo].[log_sessions]

if  exists (select * from sys.objects where object_id = object_id('[dbo].[programs]') and type in ('u'))
    drop table [dbo].[programs]

create table programs(
    _id             bigint        primary key identity,
    program_name    nvarchar(64)  not null,
    log_folder_path nvarchar(260) not null unique,
    archive_days    smallint      not null default 30,
    delete_days     smallint      not null default 90
);

create table log_sessions(
    _id           bigint primary key identity,
    program_id    bigint not null,
    has_warning   bit    null,
    has_error     bit    null,
    has_fatal     bit    null,
    created_date  date   default getdate(),
    is_archived   bit    null,

    constraint fk_program_id foreign key (program_id)
        references programs(_id)
);

create table log_entries(
    _id        bigint         primary key identity,
    session_id bigint         not null,
    log_entry  nvarchar(4000) not null,

    level as json_value([log_entry], '$.level'),
    index ix_level (level),

    constraint fk_session_id foreign key (session_id)
        references log_sessions(_id)
);
