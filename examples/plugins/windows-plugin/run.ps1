$outputFile = 'c:\tmp\results\file.txt'
$doneFile = 'c:\tmp\results\done'

Get-WinEvent > $outputFile
$outputFile | Out-File $doneFile -Encoding ascii