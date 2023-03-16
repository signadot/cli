package utils

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/cli/internal/config"
	"sigs.k8s.io/yaml"
)

var (
	/*
		Explanation with example:
		Regex pattern: @{\s*(?:([a-zA-Z][a-zA-Z0-9_.-]*)\s*:)?\s*([a-zA-Z0-9_.\\/-]*)\s*}
		String to match: "Hi! @{ embed : file.yaml } var @{ dev }!"

		// Character positions (^ below represents space in the string to match)
		// H i ! ^ @ { ^ e m b e  d  ^  :  ^  f  i  l  e  .  y  a  m  l  ^  }  ^  v  a  r  ^  @  {  ^  d  e  v  ^  }  !
		// 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39

		// ExpectedResult of `match all` regex operation:
		// [[4 26 7 12 15 24] [31 39 -1 -1 34 37]]

		// Explanation:

		// [4 26 7 12 15 24]               # directive with operation and operand
		//  4 - 26: @{^embed^:^file.yaml^} (directive string)
		//  7 - 12: embed                  (group #1: operation)
		// 15 - 24: file.yaml              (group #2: operand)

		// [31 39 -1 -1 34 37]             # plain variable substitution
		// 31 - 39: @{^dev^}               (placeholder string)
		// -1 - -1: NO_MATCH               (group #1: no operation to match)
		// 34 - 37: dev                    (group #2: variable name)
	*/
	placeholderPattern = regexp.MustCompile(`@{\s*(?:([a-zA-Z][a-zA-Z0-9_.\[\]-]*)\s*:)?\s*([a-zA-Z0-9_.\\/-]*)\s*}`)
)

type FileReaderSignature func(string) (string, error)

func ReadFileContent(filename string) (content string, err error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("error reading from file %v: %v", filename, err)
	}
	return string(b), nil
}

func LoadUnstructuredTemplate(file string, tplVals config.TemplateVals, forDelete bool, fileReader FileReaderSignature) (any, error) {
	substMap, err := substMap(tplVals)
	if err != nil {
		return nil, err
	}
	template, err := clio.LoadYAML[any](file)
	if err != nil {
		return nil, err
	}
	if forDelete {
		*template = extractName(*template)
	}
	if err := substTemplate(template, substMap, fileReader); err != nil {
		return nil, err
	}
	return template, nil
}

func extractName(rpt any) map[string]any {
	topLevel, ok := rpt.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	for k := range topLevel {
		if k == "name" {
			continue
		}
		delete(topLevel, k)
	}
	return topLevel
}

func substMap(tplVals []config.TemplateVal) (map[string]string, error) {
	substMap := map[string]string{}
	conflicts := map[string][]string{}
	for _, tv := range tplVals {
		if tVal, present := substMap[tv.Var]; present {
			if tVal != tv.Val {
				if len(conflicts[tv.Var]) == 0 {
					conflicts[tv.Var] = []string{tVal}
				}
				conflicts[tv.Var] = append(conflicts[tv.Var], tv.Val)
				continue
			}
		}
		substMap[tv.Var] = tv.Val
	}
	if len(conflicts) == 0 {
		return substMap, nil
	}
	conflictKeys := make([]string, 0, len(conflicts))
	for k := range conflicts {
		conflictKeys = append(conflictKeys, k)
	}
	sort.Strings(conflictKeys)
	msgs := make([]string, 0, len(conflictKeys))
	for _, key := range conflictKeys {
		vals := strings.Join(conflicts[key], ", ")
		msgs = append(msgs, fmt.Sprintf("%s: %s", key, vals))
	}
	return nil, fmt.Errorf("conflicting variable defs:\n\t%s", strings.Join(msgs, "\n\t"))
}

func substTemplate(rpt *any, substMap map[string]string, fileReader FileReaderSignature) error {
	vars := map[string]struct{}{}
	err := substTemplateRec(rpt, substMap, vars, fileReader)
	if err != nil {
		return err
	}
	notExpanded := []string{}
	for k := range vars {
		if _, ok := substMap[k]; !ok {
			notExpanded = append(notExpanded, k)
		}
	}
	if len(notExpanded) > 0 {
		return fmt.Errorf("unexpanded variables: %s", strings.Join(notExpanded, ", "))
	}
	return nil
}

func substTemplateRec(rpt *any, substMap map[string]string, vars map[string]struct{}, fileReader FileReaderSignature) error {
	switch x := (*rpt).(type) {
	case map[string]any:
		for k, v := range x {
			if err := substTemplateRec(&v, substMap, vars, fileReader); err != nil {
				return err
			}
			x[k] = v
		}

	case []any:
		for i := range x {
			if err := substTemplateRec(&x[i], substMap, vars, fileReader); err != nil {
				return err
			}
		}
	case string:
		res, err := substString(x, substMap, vars, fileReader)
		if err != nil {
			return err
		}
		*rpt = res
	default:
	}
	return nil
}

func substString(s string, substMap map[string]string, vars map[string]struct{}, fileReader FileReaderSignature) (any, error) {
	result := strings.Clone(s)
	matches, err := parseForMatches(s)
	if err != nil {
		return "", err
	}
	if matches == nil {
		return s, nil
	}
	for i := range matches {
		match := matches[i]
		if err != nil {
			panic(err.Error())
		}
		if match.Operation == nil { // plan variable substitution without specific operation defined in the format @{operation:operand}
			vars[match.Operand] = struct{}{}
			// plain key value substitution
			value, ok := substMap[match.Operand]
			if !ok {
				continue
			}
			result = strings.ReplaceAll(result, match.Placeholder, value)
		} else {
			// Directive based substitution
			switch *match.Operation {
			case "embed":
				filename := match.Operand
				text, err := fileReader(filename)
				if err != nil {
					return "", err
				}
				result = strings.ReplaceAll(result, match.Placeholder, text)
			case "embed[yaml]": // TODO: Handle `embed[yaml]` under `embed` later
				if match.Placeholder != s {
					return nil, errors.New("embed[yaml] directive must be a complete string. Eg. \"@{embed[yaml]:file.yaml}\" with nothing else surrounding it")
				}
				filename := match.Operand
				text, err := fileReader(filename)
				if err != nil {
					return "", err
				}
				var res interface{}
				err = yaml.Unmarshal([]byte(text), &res)
				if err != nil {
					log.Fatal(err)
				}
				return res, nil
			default:
				return "", fmt.Errorf("unsupported operation")
			}
		}
	}
	return result, nil
}

type Match struct {
	Placeholder string
	Operation   *string
	Operand     string
}

func parseForMatches(s string) ([]Match, error) {
	allMatchIndices := placeholderPattern.FindAllStringSubmatchIndex(s, -1)
	if allMatchIndices == nil {
		return nil, nil
	}
	var matches []Match
	for i := range allMatchIndices {
		singleMatchIndices := allMatchIndices[i]
		placeholder, operation, operand, err := parseByIndices(s, singleMatchIndices)
		if err != nil {
			return nil, err
		}
		match := Match{Placeholder: placeholder, Operation: operation, Operand: operand}
		matches = append(matches, match)
	}
	return matches, nil
}

func parseByIndices(str string, indices []int) (placeholder string, operation *string, operand string, err error) {
	if len(indices) != 6 {
		// Check the doc where placeholderPattern is defined on why this is expected to have 6 entries
		// This error can occur only because of faulty regex pattern in the current scenario, and
		// hence is unexpected. The caller of this function can issue a panic() when this happens.
		return "", nil, "", fmt.Errorf("unexpected regex error generating content from template")
	}

	// Here is an example showing the computation of groups based on the input string, and the
	// value for indices:

	// Input string (str) with character positions where `^` represents a space
	// H i ! ^ @ { ^ e m b e  d  ^  :  ^  f  i  l  e  .  y  a  m  l  ^  }  ^  v  a  r  ^  @  {  ^  d  e  v  ^  }  !
	// 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39

	// Consider indices = [4 26 7 12 15 24]
	//  4 - 26: @{^embed^:^file.yaml^} (directive string)
	//  7 - 12: embed                  (group #1: operation)
	// 15 - 24: file.yaml              (group #2: operand)

	placeholder = str[indices[0]:indices[1]] // matches `@{^embed^:^file.yaml%}`

	if indices[2] != -1 && indices[3] != -1 { // optional value
		x := str[indices[2]:indices[3]] // matches `embed`
		operation = &x
	}

	operand = str[indices[4]:indices[5]] // matches: `file.yaml`

	return placeholder, operation, operand, nil
}
