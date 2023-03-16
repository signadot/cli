package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/signadot/cli/internal/config"
)

const specTemplateFile = "spec-template"

type SubstTest struct {
	TestID         int
	Files          map[string]string
	Args           []string
	ExpectedError  string
	ExpectedResult string
}

// below cases hard code fields which for some reason
// are not omitempty, although in the swagger source they are.
// TODO: fix omitempties in go-sdk.
var substCases = []SubstTest{
	{
		TestID:         1,
		Files:          map[string]string{specTemplateFile: `{"name":"@{dev}-service"}`},
		Args:           []string{"dev=jane"},
		ExpectedResult: `{"name":"jane-service"}`,
	},
	{
		TestID:        2,
		Files:         map[string]string{specTemplateFile: `{"name":"@{dev}-service"}`},
		Args:          []string{"xdev=jane"},
		ExpectedError: "unexpanded variables: dev",
	},
	{
		TestID:         3,
		Files:          map[string]string{specTemplateFile: `{"name":"service-@{dev}"}`},
		Args:           []string{"dev=jane"},
		ExpectedResult: `{"name":"service-jane"}`,
	},
	{
		TestID:         4,
		Files:          map[string]string{specTemplateFile: `{"name":"@{team}-service-@{dev}"}`},
		Args:           []string{"team=gorillas", "dev=jane"},
		ExpectedResult: `{"name":"gorillas-service-jane"}`,
	},
	{
		TestID:         5,
		Files:          map[string]string{specTemplateFile: `{"name":"@{team}@{dev}"}`},
		Args:           []string{"team=gorillas", "dev=jane"},
		ExpectedResult: `{"name":"gorillasjane"}`,
	},
	{
		TestID:         6,
		Files:          map[string]string{specTemplateFile: `{"name":"b@{team}@{dev}e"}`},
		Args:           []string{"team=gorillas", "dev=jane"},
		ExpectedResult: `{"name":"bgorillasjanee"}`,
	},
	{
		TestID: 7,
		Files: map[string]string{
			specTemplateFile: `{"script1":"@{ embed : file1.py }","message":"@{ greeting } @{ name }!","script2":"@{ embed : file2.py }"}`,
			"file1.py": "#!/bin/bash\n" +
				"echo \"Seeding DB ${DBNAME}\"",
			"file2.py": "#!/bin/bash\n" +
				"echo \"Second script\"",
		},
		Args:           []string{"greeting=Hello", "name=World"},
		ExpectedResult: `{"message":"Hello World!","script1":"#!/bin/bash\necho \"Seeding DB ${DBNAME}\"","script2":"#!/bin/bash\necho \"Second script\""}`,
	},
	{
		TestID: 8,
		Files: map[string]string{
			specTemplateFile: `{"script1":"@{ embed : file1.py }","message":"@{ greeting } @{ name }!","script2":"@{ embed : file2.py }"}`,
			"file1.py": "#!/bin/bash\n" +
				"echo \"Seeding DB ${DBNAME}\"",
		},
		Args:          []string{"greeting=Hello", "name=World"},
		ExpectedError: "error reading from file: file2.py",
	},
	{
		TestID: 9,
		Files: map[string]string{
			specTemplateFile: `{"script1":"@{ embed : file1.py }"}`,
			"file1.py": "#!/bin/bash\n" +
				"echo \"Seeding DB ${DBNAME}\"",
		},
		Args:           []string{},
		ExpectedResult: `{"script1":"#!/bin/bash\necho \"Seeding DB ${DBNAME}\""}`,
	},
	{
		TestID: 10,
		Files: map[string]string{
			specTemplateFile: `{"script1":"@{ unsupported : file1.py }"}`,
			"file1.py": "#!/bin/bash\n" +
				"echo \"Seeding DB ${DBNAME}\"",
		},
		Args:          []string{},
		ExpectedError: "unsupported operation",
	},
	{
		TestID: 11,
		Files: map[string]string{
			specTemplateFile: `{"script":"@{ embed[yaml] : file1.yaml }"}`,
			"file1.yaml":     "{\"french-hens\":3,\"xmas\":true,\"calling-birds\":[\"huey\",\"dewey\"],\"pi\":3.14159,\"xmas-fifth-day\":{\"calling-birds\":\"four\"},\"doe\":\"deer\"}",
		},
		Args:           []string{},
		ExpectedResult: `{"script":{"calling-birds":["huey","dewey"],"doe":"deer","french-hens":3,"pi":3.14159,"xmas":true,"xmas-fifth-day":{"calling-birds":"four"}}}`,
	},
	{
		TestID: 12,
		Files: map[string]string{
			specTemplateFile: `{"script":"x@{ embed[yaml] : file1.yaml }y"}`,
			"file1.yaml":     "{\"french-hens\":3,\"xmas\":true,\"calling-birds\":[\"huey\",\"dewey\"],\"pi\":3.14159,\"xmas-fifth-day\":{\"calling-birds\":\"four\"},\"doe\":\"deer\"}",
		},
		Args:          []string{},
		ExpectedError: "embed[yaml] directive must be a complete string. Eg. \"@{embed[yaml]:file.yaml}\" with nothing else surrounding it",
	},
}

func testLoadUnstructuredTemplate(tc *SubstTest, t *testing.T) {
	actualFileNames := map[string]string{}
	for filename, content := range tc.Files {
		f, err := os.CreateTemp(".", filename)
		if err != nil {
			t.Error(err)
			return
		}
		defer os.Remove(f.Name())
		_, err = f.Write([]byte(content))
		if err != nil {
			t.Error(err)
			return
		}
		f.Close()
		actualFileNames[filename] = f.Name()
	}
	tplVals := &config.TemplateVals{}
	for _, arg := range tc.Args {
		if err := tplVals.Set(arg); err != nil {
			if tc.ExpectedError == "" {
				t.Errorf("unexpected error %s", err.Error())
				return
			}
			return
		}
	}
	fileReader := func(filename string) (content string, err error) {
		b, err := os.ReadFile(actualFileNames[filename])
		if err != nil {
			return "", fmt.Errorf("error reading from file: %v", filename)
		}
		return string(b), nil
	}
	template, err := LoadUnstructuredTemplate(actualFileNames[specTemplateFile], *tplVals, false, fileReader)
	if err == nil && tc.ExpectedError != "" {
		t.Errorf("error expected but not received in test #%d", tc.TestID)
		return
	} else if err != nil && tc.ExpectedError == "" {
		t.Errorf("error not expected but received in test #%d: %s", tc.TestID, err.Error())
		return
	} else if err != nil && tc.ExpectedError != "" {
		if err.Error() != tc.ExpectedError {
			t.Errorf("Test #%d: expected error %q got %q", tc.TestID, tc.ExpectedError, err.Error())
		}
		return
	}
	d, e := json.Marshal(template)
	if e != nil {
		t.Error(e)
		return
	}
	if !bytes.Equal(d, []byte(tc.ExpectedResult)) {
		t.Errorf("failed test %d: got %s want %s", tc.TestID, string(d), tc.ExpectedResult)
	}
}

func TestSubst(t *testing.T) {
	for i := range substCases {
		tc := &substCases[i]
		testLoadUnstructuredTemplate(tc, t)
	}
}
