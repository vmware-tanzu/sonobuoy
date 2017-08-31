# Local Filesystem Hook for Logrus

Sometimes developers like to write directly to a file on the filesystem. This is a hook for [logrus](https://github.com/Sirupsen/logrus) designed to allow users to do just that.  The log levels are dynamic at instanciation of the hook, so it is capable of logging at some or all levels.

## Example
```go
import (
	log "github.com/sirupsen/logrus"
	"github.com/rifflock/lfshook"
)

var Log *log.Logger

func NewLogger( config map[string]interface{} ) *log.Logger {
	if Log != nil {
		return Log
	}

	Log = log.New()
	Log.SetFormatter(&log.JSONFormatter{})
	Log.Hooks.Add(lfshook.NewHook(lfshook.PathMap{
		log.InfoLevel : "/var/log/info.log",
		log.ErrorLevel : "/var/log/error.log",
	}))
	return Log
}
```

### Formatters
lfshook will strip colors from any TextFormatter type formatters when writing to local file, because the color codes don't look so great.

### Log rotation
In order to enable automatic log rotation it's possible to provide an io.Writer instead of the path string of a log file.
In combination with pakages like [go-file-rotatelogs](https://github.com/lestrrat/go-file-rotatelogs) log rotation can easily be achieved.

```go
  import "github.com/lestrrat/go-file-rotatelogs"

  path := "/var/log/go.log"
  writer := rotatelogs.NewRotateLogs(
    path + ".%Y%m%d%H%M",
    rotatelogs.WithLinkName(path),
    rotatelogs.WithMaxAge(time.Duration(86400) * time.Second),
    rotatelogs.WithRotationTime(time.Duration(604800) * time.Second),
  )

  log.Hooks.Add(lfshook.NewHook(lfshook.WriterMap{
    logrus.InfoLevel: writer,
    logrus.ErrorLevel: writer,
  }))
```

### Note:
Whichever user is running the go application must have read/write permissions to the log files selected, or if the files do not yet exists, then to the directory in which the files will be created.
