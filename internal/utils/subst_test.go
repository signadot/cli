package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nsf/jsondiff"
	"github.com/signadot/cli/internal/config"
)

const specTemplateFile = "template.yaml"

type TestFile struct {
	Name     string
	RelPath  string
	Content  string
	FullPath string
}

type TestCase struct {
	TestName       string
	Files          []TestFile
	Args           []string
	ExpectedError  func(error) bool
	ExpectedResult string
}

// below cases hard code fields which for some reason
// are not omitempty, although in the swagger source they are.
// TODO: fix omitempties in go-sdk.
var testCases = []TestCase{
	{
		TestName: "single variable substitution (prefixed); arg available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"name":"@{dev}-service"}`,
			},
		},
		Args:           []string{"dev=jane"},
		ExpectedResult: `{"name":"jane-service"}`,
	},
	{
		TestName: "single variable substitution; arg not available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"name":"@{dev}-service"}`,
			},
		},
		Args:          []string{"xdev=jane"},
		ExpectedError: func(e error) bool { return errors.Is(e, errUnexpandedVar) },
	},
	{
		TestName: "single variable substitution (suffixed); arg available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"name":"service-@{dev}"}`,
			},
		},
		Args:           []string{"dev=jane"},
		ExpectedResult: `{"name":"service-jane"}`,
	},
	{
		TestName: "double variable substitution (prefixed and suffixed); args available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"name":"@{team}-service-@{dev}"}`,
			},
		},
		Args:           []string{"team=gorillas", "dev=jane"},
		ExpectedResult: `{"name":"gorillas-service-jane"}`,
	},
	{
		TestName: "multiple variable substitution (vars only); args available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"name":"@{team}@{dev}"}`,
			},
		},
		Args:           []string{"team=gorillas", "dev=jane"},
		ExpectedResult: `{"name":"gorillasjane"}`,
	},
	{
		TestName: "multiple variable substitution (vars in middle); args available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"name":"b@{team}@{dev}e"}`,
			},
		},
		Args:           []string{"team=gorillas", "dev=jane"},
		ExpectedResult: `{"name":"bgorillasjanee"}`,
	},
	{
		TestName: "embed from file; file available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"script1":"@{ embed : file1.py }"}`,
			},
			{
				Name:    "file1.py",
				RelPath: ".",
				Content: "#!/bin/bash\n" +
					"echo \"Seeding DB ${DBNAME}\"",
			},
		},
		Args:           []string{},
		ExpectedResult: `{"script1":"#!/bin/bash\necho \"Seeding DB ${DBNAME}\""}`,
	},
	{
		TestName: "embed from file; file not available",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"script1":"@{ embed : file1.py }"}`,
			},
			{
				Name:    "file2.py",
				RelPath: ".",
				Content: "#!/bin/bash\n" +
					"echo \"Seeding DB ${DBNAME}\"",
			},
		},
		Args:          []string{},
		ExpectedError: func(e error) bool { return os.IsNotExist(e) },
	},
	{
		TestName: "unsupported operation",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"script1":"@{ unsupported : file1.py }"}`,
			},
			{
				Name:    "file1.py",
				RelPath: ".",
				Content: "#!/bin/bash\n" +
					"echo \"Seeding DB ${DBNAME}\"",
			},
		},
		Args:          []string{},
		ExpectedError: func(e error) bool { return errors.Is(e, errUnsupportedOp) },
	},
	{
		TestName: "binary encoding embed",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"data":"@{ embed[binary] : bin }"}`,
			},
			{
				Name:    "bin",
				RelPath: ".",
				Content: string([]byte{0, 1, 100, 11, 23, 17}),
			},
		},
		Args:           []string{},
		ExpectedResult: fmt.Sprintf(`{"data": %q}`, base64.StdEncoding.EncodeToString([]byte{0, 1, 100, 11, 23, 17})),
	},
	{
		TestName: "binary encoding var",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"data":"@{ x[binary] }"}`,
			},
			{
				Name:    "bin",
				RelPath: ".",
				Content: string([]byte{0, 1, 100, 11, 23, 17}),
			},
		},
		Args:           []string{fmt.Sprintf("x=%s", string([]byte{0, 1, 100, 11, 23, 17}))},
		ExpectedResult: fmt.Sprintf(`{"data": %q}`, base64.StdEncoding.EncodeToString([]byte{0, 1, 100, 11, 23, 17})),
	},
	{
		TestName: "embed yaml; file available; embedding valid",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"script":"@{ embed[yaml] : file1.yaml }"}`,
			},
			{
				Name:    "file1.yaml",
				RelPath: ".",
				Content: "{\"french-hens\":3,\"xmas\":true,\"calling-birds\":[\"huey\",\"dewey\"],\"pi\":3.14159,\"xmas-fifth-day\":{\"calling-birds\":\"four\"},\"doe\":\"deer\"}",
			},
		},
		Args:           []string{},
		ExpectedResult: `{"script":{"calling-birds":["huey","dewey"],"doe":"deer","french-hens":3,"pi":3.14159,"xmas":true,"xmas-fifth-day":{"calling-birds":"four"}}}`,
	},
	{
		TestName: "embed yaml; file available; embedding invalid (contains more than just the directive)",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"script":"x@{ embed[yaml] : file1.yaml }"}y`,
			},
			{
				Name:    "file1.yaml",
				RelPath: ".",
				Content: "{\"french-hens\":3,\"xmas\":true,\"calling-birds\":[\"huey\",\"dewey\"],\"pi\":3.14159,\"xmas-fifth-day\":{\"calling-birds\":\"four\"},\"doe\":\"deer\"}",
			},
		},
		Args:          []string{},
		ExpectedError: func(e error) bool { return errors.Is(e, errInvalidEnc) },
	},
	{
		TestName: "embed from files in different directories",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: "dir1",
				Content: `{"script1":"@{ embed : ../file1.py }", "script2": "@{ embed : dir2/file2.py }"}`,
			},
			{
				Name:    "file1.py",
				RelPath: ".",
				Content: "#!/bin/bash\n" +
					"echo \"Seeding DB ${DBNAME}\"",
			},
			{
				Name:    "file2.py",
				RelPath: "./dir1/dir2",
				Content: "#!/bin/bash\n" +
					"echo \"Seeding DB ${DBNAME}\"",
			},
		},
		Args:           []string{},
		ExpectedResult: `{"script1":"#!/bin/bash\necho \"Seeding DB ${DBNAME}\"","script2":"#!/bin/bash\necho \"Seeding DB ${DBNAME}\""}`,
	},
	{
		TestName: "embed YAML from files in different directories",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: "dir1",
				Content: `{"first":"@{ embed[yaml] : ../file1.yaml }", "second": "@{ embed[yaml] : dir2/file2.yaml }"}`,
			},
			{
				Name:    "file1.yaml",
				RelPath: ".",
				Content: `{"calling-birds":["huey","dewey"]}`,
			},
			{
				Name:    "file2.yaml",
				RelPath: "./dir1/dir2",
				Content: `{"xmas-fifth-day":{"partridges":{"count":1}}}`,
			},
		},
		Args:           []string{},
		ExpectedResult: `{"first":{"calling-birds":["huey","dewey"]},"second":{"xmas-fifth-day":{"partridges":{"count":1}}}}`,
	},

	// Typed variable embedding. This is what lets a single --set carry a whole
	// list of forks, which the built-in sandbox template depends on; previously
	// it was covered only through the [binary] path.
	{
		TestName: "yaml encoding var; JSON list of forks",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"name":"sb","spec":{"forks":"@{forks[yaml]}"}}`,
			},
		},
		Args: []string{
			`forks=[{"forkOf":{"kind":"Deployment","namespace":"hotrod","name":"route"},` +
				`"customizations":{"images":[{"image":"acme/route:abc"}]}},` +
				`{"forkOf":{"kind":"Deployment","namespace":"hotrod","name":"frontend"}}]`,
		},
		ExpectedResult: `{"name":"sb","spec":{"forks":[` +
			`{"forkOf":{"kind":"Deployment","namespace":"hotrod","name":"route"},` +
			`"customizations":{"images":[{"image":"acme/route:abc"}]}},` +
			`{"forkOf":{"kind":"Deployment","namespace":"hotrod","name":"frontend"}}]}}`,
	},
	{
		TestName: "yaml encoding var; YAML block value",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"labels":"@{labels[yaml]}"}`,
			},
		},
		Args:           []string{"labels=a: one\nb: two\n"},
		ExpectedResult: `{"labels":{"a":"one","b":"two"}}`,
	},
	{
		// The built-in template carries a placeholder for every optional field
		// and relies on this to leave unset ones prunable.
		TestName: "yaml encoding var; empty value yields null so the key can be pruned",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"ttl":"@{ttl[yaml]}"}`,
			},
		},
		Args:           []string{"ttl="},
		ExpectedResult: `{"ttl":null}`,
	},
	{
		TestName: "yaml encoding var; scalar stays typed rather than becoming a string",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"replicas":"@{n[yaml]}"}`,
			},
		},
		Args:           []string{"n=3"},
		ExpectedResult: `{"replicas":3}`,
	},
	{
		TestName: "yaml encoding var; invalid as part of a larger string",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"forks":"x@{forks[yaml]}"}`,
			},
		},
		Args:          []string{"forks=[]"},
		ExpectedError: func(e error) bool { return errors.Is(e, errInvalidEnc) },
	},
	{
		TestName: "unrecognized encoding",
		Files: []TestFile{
			{
				Name:    specTemplateFile,
				RelPath: ".",
				Content: `{"x":"@{x[toml]}"}`,
			},
		},
		Args:          []string{"x=1"},
		ExpectedError: func(e error) bool { return e != nil },
	},
}

func testLoadUnstructuredTemplate(tc *TestCase, t *testing.T) {
	// Creating a map of file name to other information, similar to a filesystem
	fs := map[string]*TestFile{}

	// Create temporary directory with the prefix templates.
	// It creates a temp directory such as `/var/folders/nw/86cb_g4x755123f67qq0pzym0000gn/T/templates2254857160`
	dir, err := os.MkdirTemp("", "templates")
	if err != nil {
		t.Error(err)
		return
	}

	// Make additional directories to test referencing files with path.
	dir1 := filepath.Join(dir, "dir1")
	if err := os.Mkdir(dir1, 0700); err != nil {
		panic("error creating dir1 inside templates temp directory")
	}

	dir2 := filepath.Join(dir1, "dir2")
	if err := os.Mkdir(dir2, 0700); err != nil {
		panic("error creating dir2 inside dir1 directory")
	}

	defer os.RemoveAll(dir)

	// Create files under the templates temp directory
	for i := range tc.Files {
		file := &tc.Files[i]
		nameAndPath := filepath.Join(dir, file.RelPath, file.Name)

		f, err := os.Create(nameAndPath)
		if err != nil {
			t.Error(err)
			return
		}
		defer os.Remove(f.Name())
		_, err = f.Write([]byte(file.Content))
		if err != nil {
			t.Error(err)
			return
		}
		f.Close()

		file.FullPath = nameAndPath
		fs[file.Name] = file
	}

	tplVals := &config.TemplateVals{}
	for _, arg := range tc.Args {
		if err := tplVals.Set(arg); err != nil {
			if tc.ExpectedError == nil {
				t.Errorf("unexpected error %s", err.Error())
				return
			}
			return
		}
	}

	template, err := LoadUnstructuredTemplate(fs[specTemplateFile].FullPath, *tplVals, false)
	if err == nil && tc.ExpectedError != nil {
		t.Errorf("[Test: %s] error expected but not received", tc.TestName)
		return
	} else if err != nil && tc.ExpectedError == nil {
		t.Errorf("[Test: %s] error not expected but received. Error: %s", tc.TestName, err.Error())
		return
	} else if err != nil && tc.ExpectedError != nil {
		if !tc.ExpectedError(err) {
			t.Errorf("[Test: %s] unexpected got %q", tc.TestName, err.Error())
		}
		return
	}
	d, e := json.Marshal(template)
	if e != nil {
		t.Error(e)
		return
	}
	opts := jsondiff.DefaultJSONOptions()
	m, _ := jsondiff.Compare(d, []byte(tc.ExpectedResult), &opts)
	if m != jsondiff.FullMatch {
		t.Errorf("[Test: %s] got %q want %q", tc.TestName, string(d), tc.ExpectedResult)
	}
}

func TestTemplating(t *testing.T) {
	for i := range testCases {
		tc := &testCases[i]
		testLoadUnstructuredTemplate(tc, t)
	}
}

func TestRenderTemplateFromMemory(t *testing.T) {
	out, err := RenderTemplate([]byte(`{"name":"@{name}","spec":{"forks":"@{forks[yaml]}"}}`),
		config.TemplateVals{
			{Var: "name", Val: "pr-42"},
			{Var: "forks", Val: `[{"forkOf":{"name":"route"}}]`},
		}, TemplateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"pr-42","spec":{"forks":[{"forkOf":{"name":"route"}}]}}`
	opts := jsondiff.DefaultJSONOptions()
	if m, _ := jsondiff.Compare(d, []byte(want), &opts); m != jsondiff.FullMatch {
		t.Errorf("got %s want %s", d, want)
	}
}

// Embeds must resolve against BaseDir when rendering from memory, since there
// is no template path to derive a directory from.
func TestRenderTemplateEmbedUsesBaseDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "forks.yaml"), []byte(`[{"forkOf":{"name":"route"}}]`), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := RenderTemplate([]byte(`{"forks":"@{embed[yaml]: forks.yaml}"}`),
		config.TemplateVals{}, TemplateOptions{BaseDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, _ := json.Marshal(out)
	want := `{"forks":[{"forkOf":{"name":"route"}}]}`
	opts := jsondiff.DefaultJSONOptions()
	if m, _ := jsondiff.Compare(d, []byte(want), &opts); m != jsondiff.FullMatch {
		t.Errorf("got %s want %s", d, want)
	}
}

func TestConflictingVarDefs(t *testing.T) {
	_, err := RenderTemplate([]byte(`{"name":"@{x}"}`), config.TemplateVals{
		{Var: "x", Val: "one"},
		{Var: "x", Val: "two"},
	}, TemplateOptions{})
	if err == nil {
		t.Fatal("expected an error for conflicting definitions")
	}
	if !strings.Contains(err.Error(), "conflicting variable defs") {
		t.Errorf("unexpected error: %v", err)
	}
}

// Repeating the same binding with the same value is not a conflict.
func TestRepeatedIdenticalVarDef(t *testing.T) {
	out, err := RenderTemplate([]byte(`{"name":"@{x}"}`), config.TemplateVals{
		{Var: "x", Val: "one"},
		{Var: "x", Val: "one"},
	}, TemplateOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, _ := json.Marshal(out)
	if string(d) != `{"name":"one"}` {
		t.Errorf("got %s", d)
	}
}

func TestUnexpandedVarsAreSorted(t *testing.T) {
	_, err := RenderTemplate([]byte(`{"a":"@{zulu}","b":"@{alpha}"}`),
		config.TemplateVals{}, TemplateOptions{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "unexpanded variable: alpha, zulu"; err.Error() != want {
		t.Errorf("got %q want %q", err.Error(), want)
	}
}

func TestUnstructuredToNameAndSpec(t *testing.T) {
	name, spec, err := UnstructuredToNameAndSpec(map[string]any{
		"name": "sb",
		"spec": map[string]any{"cluster": "demo"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "sb" {
		t.Errorf("got name %q", name)
	}
	if spec == nil {
		t.Error("expected a spec")
	}

	if _, _, err := UnstructuredToNameAndSpec(map[string]any{"spec": map[string]any{}}); err == nil {
		t.Error("expected an error when name is missing")
	}
}
