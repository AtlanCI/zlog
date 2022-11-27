package test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/AtlanCI/zlog"
	"github.com/AtlanCI/zlog/helper"
)

func TestMain(m *testing.M) {
	l, err := helper.NewTextFileLogger("./test.log")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "NewTextFileLogger: ", err)
		return
	}

	// stdout
	//l := helper.NewTextStdoutLogger()

	zlog.AddLoggers(l)
	zlog.SetCallerEnable(true)
	defer zlog.Close()

	m.Run()
}

func TestDebug(t *testing.T) {
	zlog.Debugf(context.TODO(), "this is a debug log. a=%d", 888)
}

func TestInfo(t *testing.T) {
	zlog.Infof(context.TODO(), "this is a info log. a=%d", 888)
}

func TestWarn(t *testing.T) {
	zlog.Warnf(context.TODO(), "this is a warn log. a=%d", 888)
}

func TestError(t *testing.T) {
	zlog.Errorf(context.TODO(), "this is a error log. a=%d", 888)
}
