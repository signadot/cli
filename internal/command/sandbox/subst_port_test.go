package sandbox

import (
	"testing"

	"sigs.k8s.io/yaml"
)

func TestUnstructuredToSandboxPorts(t *testing.T) {
	const doc = `
name: sb
spec:
  cluster: c
  labels:
    port: "8080"
  resources:
  - name: db
    plugin: mysql
    params:
      port: "3306"
  forks:
  - forkOf:
      kind: Deployment
      namespace: ns
      name: app
    endpoints:
    - name: http
      port: "8080"
      protocol: http
`
	var un any
	if err := yaml.Unmarshal([]byte(doc), &un); err != nil {
		t.Fatal(err)
	}
	sb, err := unstructuredToSandbox(un)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sb.Spec.Resources[0].Params["port"]; got != "3306" {
		t.Errorf("resource param port = %q, want \"3306\"", got)
	}
	if got := sb.Spec.Labels["port"]; got != "8080" {
		t.Errorf("label port = %q, want \"8080\"", got)
	}
	if got := sb.Spec.Forks[0].Endpoints[0].Port; got != 8080 {
		t.Errorf("fork endpoint port = %d, want 8080", got)
	}
}
