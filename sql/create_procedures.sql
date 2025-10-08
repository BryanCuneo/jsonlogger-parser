create procedure archive_logs
as begin
    set nocount on;

    update log_sessions
    set is_archived = 1
    from log_sessions
    inner join programs on programs._id = log_sessions.program_id
    where (log_sessions.is_archived is null or log_sessions.is_archived = 0)
        and log_sessions.created_date < dateadd(day, -programs.archive_days, getdate());
end;