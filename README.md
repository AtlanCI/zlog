# zlog

a simple async log library of go

# Example

first `go get`

```shell
go get github.com/AtlanCI/zlog
```

## Write file

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AtlanCI/zlog"
	"github.com/AtlanCI/zlog/helper"
)

func main() {
	l, err := helper.NewTextFileLogger("./test.log")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "NewTextFileLogger: ", err)
		return
	}

	zlog.AddLoggers(l)

	// Enabled on Caller will reduce performance by 15%
	zlog.SetCallerEnable(true)

	defer zlog.Close()

	zlog.Debugf(context.TODO(), "this is a debug log. a=%d", 888)
}

```
## Stdout
```go
package main

import (
	"context"

	"github.com/AtlanCI/zlog"
	"github.com/AtlanCI/zlog/helper"
)

func main() {

	// stdout 
	l := helper.NewTextStdoutLogger()
	
	zlog.AddLoggers(l)

	// Enabled on Caller will reduce performance by 15%
	zlog.SetCallerEnable(true)

	defer zlog.Close()

	zlog.Debugf(context.TODO(), "this is a debug log. a=%d", 888)
}
```
## Feat

- [x] scroll file

- [x] trace id

- [x] Asynchronous

## todo

- [ ] stained

- [ ] compression

- [ ] more test

- [ ] delete old file

- [ ] document

- [ ] godoc and awesome go