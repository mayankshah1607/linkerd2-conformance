package stat

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/linkerd/linkerd2-conformance/utils"
	"github.com/linkerd/linkerd2/testutil"
	"github.com/onsi/gomega"
)

var emojivotoNs = "emojivoto"

type testCase struct {
	args         []string
	expectedRows map[string]string
}

var testCases = []testCase{
	{
		args: []string{"stat", "deploy", "-n", emojivotoNs},
		expectedRows: map[string]string{
			"emoji":    "1/1",
			"vote-bot": "1/1",
			"voting":   "1/1",
			"web":      "1/1",
		},
	},
	{
		args: []string{"stat", "deploy", "-n", emojivotoNs, "--to", "deploy/emoji"},
		expectedRows: map[string]string{
			"web": "1/1",
		},
	},
	{
		args: []string{"stat", "deploy", "-n", emojivotoNs, "--from", "deploy/web"},
		expectedRows: map[string]string{
			"emoji":  "1/1",
			"voting": "1/1",
		},
	},
	{
		args: []string{"stat", "deploy", "-n", emojivotoNs, "--to", "svc/emoji-svc"},
		expectedRows: map[string]string{
			"web": "1/1",
		},
	},
	{
		args: []string{"stat", "deploy", "-n", emojivotoNs, "--to", "svc/voting-svc"},
		expectedRows: map[string]string{
			"web": "1/1",
		},
	},
	{
		args: []string{"stat", "deploy", "-n", emojivotoNs, "--to", "svc/web-svc"},
		expectedRows: map[string]string{
			"vote-bot": "1/1",
		},
	},
	{
		args: []string{"stat", "ns", emojivotoNs},
		expectedRows: map[string]string{
			"emojivoto": "4/4",
		},
	},
}

func testStat(tc testCase) {
	h, _ := utils.GetHelperAndConfig()
	timeout := time.Second * 20
	err := h.RetryFor(timeout, func() error {
		tc.args = append(tc.args, "-t", "30s")
		out, stderr, err := h.LinkerdRun(tc.args...)
		if err != nil {
			return fmt.Errorf("failed to run `stat`: %s\n%s", stderr, out)
		}
		expectedColumnCount := 8

		rowStats, err := testutil.ParseRows(out, len(tc.expectedRows), expectedColumnCount)
		if err != nil {
			return fmt.Errorf("failed to parse rows: %s", err.Error())
		}
		for name, meshed := range tc.expectedRows {
			if err := validateRowStats(name, meshed, rowStats); err != nil {
				return err
			}
		}
		return nil
	})

	gomega.Expect(err).Should(gomega.BeNil(), utils.Err(err))
}

func validateRowStats(name, expectedMeshCount string, rowStats map[string]*testutil.RowStat) error {
	if name == "vote-bot" { // ignore vote-bot
		return nil
	}

	stat, ok := rowStats[name]
	if !ok {
		return fmt.Errorf("No stats found for [%s]", name)

	}

	if stat.Meshed != expectedMeshCount {
		return fmt.Errorf("Expected mesh count [%s] for [%s], got [%s]",
			expectedMeshCount, name, stat.Meshed)

	}

	if !strings.HasSuffix(stat.Rps, "rps") {
		return fmt.Errorf("Unexpected rps for [%s], got [%s]",
			name, stat.Rps)

	}

	if !strings.HasSuffix(stat.P50Latency, "ms") {
		return fmt.Errorf("Unexpected p50 latency for [%s], got [%s]",
			name, stat.P50Latency)

	}

	if !strings.HasSuffix(stat.P95Latency, "ms") {
		return fmt.Errorf("Unexpected p95 latency for [%s], got [%s]",
			name, stat.P95Latency)

	}

	if !strings.HasSuffix(stat.P99Latency, "ms") {
		return fmt.Errorf("Unexpected p99 latency for [%s], got [%s]",
			name, stat.P99Latency)

	}

	if stat.TCPOpenConnections != "-" {
		_, err := strconv.Atoi(stat.TCPOpenConnections)
		if err != nil {
			return fmt.Errorf("Error parsing number of TCP connections [%s]: %s", stat.TCPOpenConnections, err.Error())

		}

	}

	return nil

}
