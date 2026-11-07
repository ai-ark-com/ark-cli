package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/ai-ark-com/ark-cli/internal/client"
	"github.com/ai-ark-com/ark-cli/internal/output"
)

func TestPresentSummarisesCreditsAndLimits(t *testing.T) {
	res := &client.Result{
		Body:   json.RawMessage(`{"content":[{"id":"1"}],"totalElements":7}`),
		Credit: "-0.2",
		Limits: []client.Limit{
			{Window: time.Second, Max: 10, Remaining: 9, Reset: time.Second},
			{Window: 4 * time.Hour, Max: 4000, Remaining: 3660, Reset: 62 * time.Minute},
			{Window: 7 * 24 * time.Hour, Max: 25000, Remaining: 24090, Reset: -1},
		},
	}
	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := present(cmd, res, output.Options{Format: output.Table}); err != nil {
		t.Fatal(err)
	}
	want := "\n7 total matches, 0.2 credits used\n" +
		"limits: 340 of 4000 requests per 4 hours used (resets in 1h 2m); 910 of 25000 requests per 7 days used\n"
	if stderr.String() != want {
		t.Errorf("summary:\n got %q\nwant %q", stderr.String(), want)
	}

	stderr.Reset()
	unlimited := &client.Result{Body: res.Body, Limits: res.Limits[1:2]}
	if err := present(cmd, unlimited, output.Options{Format: output.Table}); err != nil {
		t.Fatal(err)
	}
	want = "\n7 total matches, 0 credits used\nlimits: 340 of 4000 requests per 4 hours used (resets in 1h 2m)\n"
	if stderr.String() != want {
		t.Errorf("summary of a call that was not charged:\n got %q\nwant %q", stderr.String(), want)
	}

	stderr.Reset()
	if err := present(cmd, res, output.Options{Format: output.JSON}); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Errorf("json output must stay free of a summary, got %q", stderr.String())
	}
}
