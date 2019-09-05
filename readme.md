# gol
Logging utility with a few goodies:
<ul>
<li/> Asynchronous (requests don't do disk IOs for logging)
<li/> Log rotation (gol will create new log file once a configurable size is reached)
<li/> Purge log files that are older than a configurable number of days
<li/> (Hopefully) sensible default values for all parameters
<li/> Easy interface for logging public access (a la Apache Web Server)
</ul>

## Usage

```
import "github.com/deal/gol"

gol.SetAppLogFolder("/path/to/log/folder")     // Log folder for service log (default /var/log)
gol.SetPublicLogFolder("/path/to/log/folder")  // Log folder for public access log (default /var/log)
gol.SetPublicLogMaxSize(200)  // Maximum size of a log file in MB
gol.SetPublicLogMaxAge(20)    // Max age of a file before it's being purged in days (default 10 days)
gol.LogToStdout(true)         // Also log to stdout  (default true)
gol.ShowLineNumbers(false)    // Show file name and line number (default false)

gol.start()  // Start gol (typically in the init() method of the main file of a service)

gol.SetAppLogLevel(gol.INFO)  // Set the logging level (default INFO)

gol.Debug("my message")   // logs a debug message (async)
gol.Info("my message")    // logs an info message (async)
gol.Warn("my message")    // logs a warning message (async)
gol.Error("my message")   // logs an error message (async)
gol.Fatal("my message")   // *synchronously* logs a fatal message and exit with code 1

go.Public(myRequest)  // Logs info about the http request and response (Apache web server style)

gol.Stop()  // stops gol (typically during graceful shutdown of the service.)
```

## Log file names

Service log files and public access log files will look like this:
```
alex@ip-192-168-1-14:~/work/deal/dev5/src/shorty/logs --> ls -alh
-rw-r--r--   1 alex  staff   100MB Aug 17 19:52 APP-LOG-2017-08-18-0
-rw-r--r--   1 alex  staff   100MB Aug 17 19:55 APP-LOG-2017-08-18-1
-rw-r--r--   1 alex  staff   100MB Aug 17 20:05 APP-LOG-2017-08-18-2
-rw-r--r--   1 alex  staff   100MB Aug 17 20:09 PUBLIC-ACCESS-LOG-2017-08-18-0
-rw-r--r--   1 alex  staff   100MB Aug 17 20:12 PUBLIC-ACCESS-LOG-22017-08-18-1
-rw-r--r--   1 alex  staff   100MB Aug 17 20:13 PUBLIC-ACCESS-LOG-22017-08-18-2
```

