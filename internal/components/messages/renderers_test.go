package messages

import (
	"fmt"
	"sync"
	"testing"

	"github.com/weave-agent/weave/sdk"
)

type testMessageRenderer struct{}

func (testMessageRenderer) Render(content string, _ sdk.ThemeInfo, _ int) string {
	return content
}

func TestMessageRenderers_ConcurrentAccess(t *testing.T) {
	ResetMessageRenderers()
	t.Cleanup(ResetMessageRenderers)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(2)

		go func(i int) {
			defer wg.Done()

			SetMessageRenderer(fmt.Sprintf("type-%d", i), testMessageRenderer{})
		}(i)

		go func(i int) {
			defer wg.Done()

			_, _ = GetMessageRenderer(fmt.Sprintf("type-%d", i))
		}(i)
	}

	wg.Wait()
}
