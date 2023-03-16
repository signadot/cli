package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type ParsePlaceholderTest struct {
	TestName       string
	Input          string
	ExpectedResult []Match
	ExpectedError  *string
}

var op = "op"
var parserTests = []ParsePlaceholderTest{
	{
		TestName:       "directive without spaces",
		Input:          "x@{op:q}y",
		ExpectedResult: []Match{{"@{op:q}", &op, "q"}},
	},
	{
		TestName:       "directive with spaces",
		Input:          "x@{ op  :   q    }y",
		ExpectedResult: []Match{{"@{ op  :   q    }", &op, "q"}},
	},
	{
		TestName:       "Variable substitution without spaces",
		Input:          "x@{dev}y",
		ExpectedResult: []Match{{"@{dev}", nil, "dev"}},
	},
	{
		TestName:       "Variable substitution with spaces",
		Input:          "x@{ dev  }y",
		ExpectedResult: []Match{{"@{ dev  }", nil, "dev"}},
	},
}

func testPlaceholderParser(tc *ParsePlaceholderTest, t *testing.T) {
	matches, err := parseForMatches(tc.Input)
	if err == nil && tc.ExpectedError != nil {
		t.Errorf("[%s] error expected but not received", tc.TestName)
		return
	} else if err != nil && tc.ExpectedError == nil {
		t.Errorf("[%s] error not expected but received: %s", tc.TestName, err.Error())
		return
	} else if err != nil && tc.ExpectedError != nil {
		if err.Error() != *tc.ExpectedError {
			t.Errorf("[%s] expected error %q got %q", tc.TestName, *tc.ExpectedError, err.Error())
		}
		return
	}
	if !cmp.Equal(matches, tc.ExpectedResult) {
		t.Errorf("[%s] failed: got %v want %v", tc.TestName, matches, tc.ExpectedResult)
	}
}

func TestPlaceholderParser(t *testing.T) {
	for i := range parserTests {
		in := &parserTests[i]
		testPlaceholderParser(in, t)
	}
}
