package bench

import "testing"

func TestCountVulns(t *testing.T) {
	report := `{"SchemaVersion":2,"Results":[
		{"Target":"alpine","Vulnerabilities":[{"VulnerabilityID":"CVE-1"},{"VulnerabilityID":"CVE-2"}]},
		{"Target":"app","Vulnerabilities":null},
		{"Target":"node","Vulnerabilities":[{"VulnerabilityID":"CVE-3"}]}
	]}`
	n, err := countVulns([]byte(report))
	if err != nil || n != 3 {
		t.Fatalf("got %d, %v", n, err)
	}
	if _, err := countVulns([]byte("not json")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestTail(t *testing.T) {
	if got := tail("a\nb\nc\n", 2); got != "b\nc" {
		t.Fatalf("got %q", got)
	}
}
