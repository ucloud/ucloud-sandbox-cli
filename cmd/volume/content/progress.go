package content

import (
	"io"
	"time"

	"github.com/schollz/progressbar/v3"
)

func newTransferProgress(out io.Writer, size int64, description string) *progressbar.ProgressBar {
	if size <= 0 {
		size = -1
	}
	return progressbar.NewOptions64(size,
		progressbar.OptionSetWriter(out),
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(20),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionUseIECUnits(true),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSetSpinnerChangeInterval(0),
		progressbar.OptionSetRenderBlankState(true),
	)
}
