Import-Module ps-jsonlogger -Force

$weightedLevels = @(
    "INFO", "INFO", "INFO", "INFO", "INFO",
    "WARNING", "WARNING",
    "ERROR",
    "DEBUG", "DEBUG", "DEBUG",
    "VERBOSE"
)

Get-Content "./jsonlogger-parser/.env" | ForEach-Object {
    $name, $value = $_.split('=')
    Set-Content env:\$name $value
}

$current_date = Get-Date

1..$env:TEST_DATA_NUM_PROGRAMS | ForEach-Object -Parallel {
    $program_num = $_
    $program_name = "Program Number $program_num"

    $folder_path = Join-Path -Path (Get-Location) -ChildPath (Join-Path -Path "ignore\sample_logs\" -ChildPath $($program_name -replace " ", ""))
    if (-not (Test-Path -Path $folder_path -PathType Container)) {
        New-Item -Path $folder_path -ItemType Directory | Out-Null
    }

    1..$env:TEST_DATA_NUM_DAYS | ForEach-Object {
        $day_num = $_
        $datestamp = (Get-Date $using:current_date).AddDays(-$day_num + 1).ToString("yyyyMMdd")
        $log_file_path = Join-Path $folder_path "$datestamp.log"
        $L = New-JsonLogger -LogFilePath $log_file_path -ProgramName $program_name -Overwrite
        (Get-ChildItem -Path $log_file_path).CreationTime = (Get-Date).AddDays(-$day_num + 1)

        1..$env:TEST_DATA_NUM_ENTRIES | ForEach-Object {
            $level = $using:weightedLevels | Get-Random

            # Skip WARNING or ERROR levels for some days
            if ( -not (
                    ($day_num % 3 -eq 0 -and $level -like "ERROR") -or
                    ($day_num % 5 -eq 0 -and $level -like "WARNING")
                )) {
                $L.Log($level, "Test $level message")
            }

            # Log("FATAL") on last log file
            if ($program_num -eq $env:TEST_DATA_NUM_PROGRAMS -and
                $day_num -eq $env:TEST_DATA_NUM_DAYS -and
                $_ -eq $env:TEST_DATA_NUM_ENTRIES
            ) {
                $L.Log("FATAL", "End of test data creation.")
            }
        }

        if ($program_num % 2 -eq 0) {
            $L.Close("All done!")
        }
    }
}