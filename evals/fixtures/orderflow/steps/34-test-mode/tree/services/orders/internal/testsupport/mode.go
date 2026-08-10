package testsupport

import (
	"fmt"
	"os"
	"testing"
)

// ModeEnv names the store backend the orders suite runs against.
const ModeEnv = "ORDERFLOW_TEST_MODE"

// Supported backends, in the order we added them.
const (
	ModeMemory   = "memory"
	ModeJSONFile = "jsonfile"
)

// Main is the body of TestMain in every package of the orders service. The
// suite refuses to guess a backend: a run against the wrong one passes for
// reasons that have nothing to do with the change under test.
func Main(m *testing.M) {
	switch mode := os.Getenv(ModeEnv); mode {
	case ModeMemory, ModeJSONFile:
		os.Exit(m.Run())
	case "":
		fmt.Fprintf(os.Stderr, "%s is not set: choose a store backend, %s or %s\n",
			ModeEnv, ModeMemory, ModeJSONFile)
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "%s=%q is not a store backend, use %s or %s\n",
			ModeEnv, mode, ModeMemory, ModeJSONFile)
		os.Exit(1)
	}
}

// Mode returns the backend the suite was started with.
func Mode() string {
	return os.Getenv(ModeEnv)
}
