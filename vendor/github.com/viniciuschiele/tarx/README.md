tarx
=================================
tarx is a golang package for archiving files and folders to tar format.

### Installation
    $ go get github.com/viniciuschiele/tarx

### Examples
Compressing a file.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    if err := tarx.Compress("example.tar", "example.txt", nil); err != nil {
        panic(err)
    }
}
```

Compressing a folder.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    if err := tarx.Compress("example.tar", "example/folder", nil); err != nil {
      panic(err)
    }
}
```

Compression a folder with Gzip algorithm.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    err := tarx.Compress("example.tar.gz", "example/folder", &tarx.CompressOptions{Compression: tarx.Gzip})
    if err != nil {
        panic(err)
    }
}
```

Keeping the last directory name in the path.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    err := tarx.Compress("example.tar", "example/folder", &tarx.CompressOptions{IncludeSourceDir: true})
    if err != nil {
      return err
    }
}
```

Filtering files to be compressed.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    filters := []string{"a.txt", "c/c2.txt"}
    err := tarx.Compress("example.tar", "example/folder", &tarx.CompressOptions{filters: filters})
    if eer != nil {
        panic(err)
    }
}
```

Extracting tar file into a directory.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    if err := tarx.Extract("example.tar", "outputDir", nil}); err != nil {
        panic(err)
    }
}
```

Extracting tar file into a directory with filters.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    filters := []string{"a.txt", "c/c2.txt"}
    err := tarx.Extract("example.tar", "outputDir", &tarx.ExtractOptions{Filters: filters})
    if err != nil {
        panic(err)
    }
}
```

Reading a specific file from the tar file.

```go
package main

import "github.com/viniciuschiele/tarx"

func main() {
    header, reader, err := tarx.Find("example.tar", "a.txt")
    if err != nil {
        panic(err)
    }
}
```
