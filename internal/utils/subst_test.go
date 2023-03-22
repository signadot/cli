package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
