package sandbox

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/signadot/cli/internal/config"
)

type SubstTest struct {
	Yaml   string
	Args   []string
	Error  bool
	Result string
}

// below cases hard code fields which for some reason
// are not omitempty, although in the swagger source they are.
// TODO: fix omitempties in go-sdk.
var substCases = []SubstTest{
	{
		Yaml:   `{"name":"@{dev}-service"}`,
		Args:   []string{"dev=jane"},
		Result: `{"endpoints":null,"name":"jane-service"}`,
	},
	{
		Yaml:  `{"name":"@{dev}-service"}`,
		Args:  []string{"xdev=jane"},
		Error: true,
	},
	{
		Yaml:   `{"name":"service-@{dev}"}`,
		Args:   []string{"dev=jane"},
		Result: `{"endpoints":null,"name":"service-jane"}`,
	},
	{
		Yaml:   `{"name":"@{team}-service-@{dev}"}`,
		Args:   []string{"team=gorillas", "dev=jane"},
		Result: `{"endpoints":null,"name":"gorillas-service-jane"}`,
	},
	{
		Yaml:   `{"name":"@{team}@{dev}"}`,
		Args:   []string{"team=gorillas", "dev=jane"},
		Result: `{"endpoints":null,"name":"gorillasjane"}`,
	},
	{
		Yaml:   `{"name":"b@{team}@{dev}e"}`,
		Args:   []string{"team=gorillas", "dev=jane"},
		Result: `{"endpoints":null,"name":"bgorillasjanee"}`,
	},
	{
		Yaml:   `{"name":"aaa","spec":{"cluster":"@{dev}-cluster"}}`,
		Args:   []string{"dev=jane"},
		Result: `{"endpoints":null,"name":"aaa","spec":{"cluster":"jane-cluster","endpoints":null,"forks":null,"local":null,"resources":null,"virtual":null}}`,
	},
}

func testSubstCase(tc *SubstTest, t *testing.T) {
	f, err := os.CreateTemp(".", "substCase")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.Remove(f.Name())
	_, err = f.Write([]byte(tc.Yaml))
	if err != nil {
		t.Error(err)
		return
	}
	f.Close()
	tplVals := &config.TemplateVals{}
	for _, arg := range tc.Args {
		if err := tplVals.Set(arg); err != nil {
			if !tc.Error {
				t.Errorf("unexpected error %s", err.Error())
				return
			}
			return
		}
	}
	sb, err := loadSandbox(f.Name(), *tplVals, false)
	if err == nil && tc.Error {
		t.Errorf("didn't get error on %s", tc.Yaml)
		return
	}
	if err != nil {
		if !tc.Error {
			t.Errorf("error processing yaml %s: %s", tc.Yaml, err.Error())
		}
		return
	}
	d, e := json.Marshal(sb)
	if e != nil {
		t.Error(e)
		return
	}
	if !bytes.Equal(d, []byte(tc.Result)) {
		t.Errorf("unexpected got %s want %s", string(d), tc.Result)
	}
}

func TestSubst(t *testing.T) {
	for i := range substCases {
		tc := &substCases[i]
		testSubstCase(tc, t)
	}
}
