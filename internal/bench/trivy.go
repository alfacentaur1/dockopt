package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Trivy counts vulnerabilities with the trivy CLI.
type Trivy struct {
	bin string
}

// NewTrivy finds trivy on PATH.
func NewTrivy() (*Trivy, error) {
	bin, err := exec.LookPath("trivy")
	if err != nil {
		return nil, fmt.Errorf("--trivy needs trivy on PATH: %w", err)
	}
	return &Trivy{bin: bin}, nil
}

// CVEs scans a local image and returns the number of vulnerabilities.
func (t *Trivy) CVEs(ctx context.Context, tag string) (int, error) {
	var out, errOut bytes.Buffer
	cmd := exec.CommandContext(ctx, t.bin, "image", "--quiet", "--format", "json", "--scanners", "vuln", tag)
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("%w: %s", err, tail(errOut.String(), 10))
	}
	return countVulns(out.Bytes())
}

// countVulns sums vulnerabilities over all results of a trivy JSON report.
func countVulns(report []byte) (int, error) {
	var r struct {
		Results []struct {
			Vulnerabilities []json.RawMessage `json:"Vulnerabilities"`
		} `json:"Results"`
	}
	if err := json.Unmarshal(report, &r); err != nil {
		return 0, fmt.Errorf("can't read trivy output: %w", err)
	}
	n := 0
	for _, res := range r.Results {
		n += len(res.Vulnerabilities)
	}
	return n, nil
}
