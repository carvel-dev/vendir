package directory

import (
	"strings"
	"sync"

	"github.com/cppforlife/go-cli-ui/ui"
)

const (
	infoLogPrefix = "  "
)

type InfoLog struct {
	ui        ui.UI
	indent    bool
	writeLock sync.RWMutex
}

func NewInfoLog(ui ui.UI) *InfoLog {
	return &InfoLog{ui: ui, indent: true}
}

func (l *InfoLog) Write(data []byte) (int, error) {
	l.writeLock.Lock()
	defer l.writeLock.Unlock()

	dataStr := string(data)

	if l.indent {
		l.indent = false
		dataStr = infoLogPrefix + dataStr
	}

	strippedLastNl := false
	if strings.HasSuffix(dataStr, "\n") {
		l.indent = true
		strippedLastNl = true
		dataStr = dataStr[:len(dataStr)-1]
	}
	dataStr = strings.Replace(dataStr, "\n", "\n"+infoLogPrefix, -1)
	if strippedLastNl {
		dataStr += "\n"
	}

	l.ui.BeginLinef("%s", dataStr)

	return len(data), nil
}
