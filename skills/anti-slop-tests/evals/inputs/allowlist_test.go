package allowlist

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowsExactHost(t *testing.T) {
	a := New([]string{"example.com"})
	require.True(t, a.Allows("example.com"))
}

func TestRejectsSuffixLookalike(t *testing.T) {
	a := New([]string{"example.com"})
	require.False(t, a.Allows("notexample.com"))
}

func TestEmptyAllowlistRejectsEverything(t *testing.T) {
	a := New(nil)
	require.False(t, a.Allows("example.com"))
}

func TestTrailingDotIsNormalized(t *testing.T) {
	a := New([]string{"example.com"})
	require.True(t, a.Allows("example.com."))
}

func TestAuditRecordWrittenOnRejection(t *testing.T) {
	audit := &fakeAudit{}
	a := New([]string{"example.com"}, WithAudit(audit))

	a.Allows("evil.test")

	require.Equal(t, []string{"evil.test"}, audit.rejected)
}

func TestPunycodeHostDoesNotBypassAllowlist(t *testing.T) {
	a := New([]string{"example.com"})
	require.False(t, a.Allows("xn--exmple-cua.com"))
}

func TestResolveReturnsErrorForMalformedHost(t *testing.T) {
	a := New([]string{"example.com"})
	_, err := a.Resolve(context.Background(), "http://[::1")
	require.Error(t, err)
}

func TestRoundTripPreservesEntries(t *testing.T) {
	a := New([]string{"a.test", "b.test"})
	got := New(a.Entries())
	require.Equal(t, a.Entries(), got.Entries())
}
