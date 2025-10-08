Import-Module ps-jsonlogger

$weightedLevels = @(
    "INFO", "INFO", "INFO", "INFO", "INFO",
    "WARNING", "WARNING",
    "ERROR",
    "DEBUG", "DEBUG", "DEBUG",
    "VERBOSE"
)

$num_programs = 5
$num_days = 60
$num_entries = 100
$current_date = Get-Date

1..$num_programs | ForEach-Object -Parallel {
    $program_num = $_
    $program_name = "Program Number $program_num"

    $folder_path = Join-Path -Path (Get-Location) -ChildPath (Join-Path -Path "ignore\sample_logs\" -ChildPath $($program_name -replace " ", ""))
    if (-not (Test-Path -Path $folder_path -PathType Container)) {
        New-Item -Path $folder_path -ItemType Directory | Out-Null
    }

    1..$using:num_days | ForEach-Object {
        $datestamp = (Get-Date $using:current_date).AddDays(-$_ + 1).ToString("yyyyMMdd")
        $L = New-JsonLogger -LogFilePath (Join-Path $folder_path -ChildPath "$datestamp.log") -ProgramName $program_name -Overwrite

        1..$using:num_entries | ForEach-Object {
            $level = $using:weightedLevels | Get-Random
            $L.Log($level, "Test $level message")
        }

        if ($program_num % 2 -eq 0) {
            $L.Close("All done!")
        }
    }
}