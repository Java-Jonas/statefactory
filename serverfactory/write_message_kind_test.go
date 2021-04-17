package serverfactory

import (
	"bar-cli/testutils"
	"strings"
	"testing"
)

func TestWriteMessageKinds(t *testing.T) {
	t.Run("writes message kinds", func(t *testing.T) {
		sf := newServerFactory(newSimpleASTExample())
		sf.writeMessageKinds()

		actual := testutils.FormatCode(sf.buf.String())
		expected := testutils.FormatCode(strings.Join([]string{
			messageKindAction_addItemToPlayer_type,
		}, "\n"))

		if expected != actual {
			t.Errorf(testutils.Diff(actual, expected))
		}
	})
}
