package rest

import (
	"testing"
	"fmt"
)

func TestIsHttpMethodBodyable_when_parameterIsEmptyString(t *testing.T) {
	// GIVEN
	var emptyString string

	// WHEN
	actual := isHttpMethodBodyable(emptyString)

	// THEN
	if actual == true {
		t.Errorf("Actual: '%t', expected: '%t'.", actual, false)
	}
}

func TestIsHttpMethodBodyable_when_parameterIsGet(t *testing.T) {
	// GIVEN
	var getValue string = "GET"

	// WHEN
	actual := isHttpMethodBodyable(getValue)

	// THEN
	if actual == true {
		t.Errorf("Actual: '%t', expected: '%t'.", actual, false)
	}
}

func TestIsHttpMethodBodyable_when_parameterIsPost(t *testing.T) {
	// GIVEN
	var postValue string = "POST"

	// WHEN
	actual := isHttpMethodBodyable(postValue)

	// THEN
	if actual == false {
		t.Errorf("Actual: '%t', expected: '%t'.", actual, false)
	}
}

func TestIsValidPath_when_error_parameterIsEmptyString(t *testing.T) {
	// GIVEN
	var emptyString string

	// WHEN
	actual, err := isValidPath(emptyString)

	// THEN
	if actual == true {
		if err == nil {
			t.Errorf("Actual: '%t', expected: '%t'", actual, false)
		} else {
			t.Errorf("Actual: '%t', expected: '%t', error: '%s'", actual, false, err.Error())
		}
	}
}

func TestIsValidPath_when_error_parameterIsWrongValue(t *testing.T) {
	// GIVEN
	var path string = "/{}"

	// WHEN
	actual, err := isValidPath(path)

	// THEN
	if actual == true {
		if err == nil {
			t.Errorf("Actual: '%t', expected: '%t'", actual, false)
		} else {
			t.Errorf("Actual: '%t', expected: '%t', error: '%s'", actual, false, err.Error())
		}
	}
}

func TestIsValidPath_when_nominal_simpleSlash(t *testing.T) {
	// GIVEN
	var path string = "/"

	// WHEN
	actual, err := isValidPath(path)

	// THEN
	if actual == false {
		if err == nil {
			t.Errorf("Actual: '%t', expected: '%t'", actual, true)
		} else {
			t.Errorf("Actual: '%t', expected: '%t', error: '%s'", actual, true, err.Error())
		}
	}
}

func TestIsValidPath_when_nominal_complexValue(t *testing.T) {
	// GIVEN
	var path string = "/a/{mo-ck1}/bbb/{m-o-ck2}/a-b-c1/{mock3}"

	// WHEN
	actual, err := isValidPath(path)

	// THEN
	if actual == false {
		if err == nil {
			t.Errorf("Actual: '%t', expected: '%t'", actual, true)
		} else {
			t.Errorf("Actual: '%t', expected: '%t', error: '%s'", actual, true, err.Error())
		}
	}
}

func TestExtractPathVariableNames_when_nominal(t *testing.T) {
	// GIVEN
	var path string = "/a/{mo-ck1}/bbb/{m-o-ck2}/a-b-c1/{mock3}"

	// WHEN
	v := extractPathVariableNames(path)

	// THEN
	if len(v) != 3 {
		t.Errorf("Actual: '%d', expected: '%d'", len(v), 3)
	} else {
		if v[0].pathIndex != 1 || v[0].variableName != "mo-ck1" {
			t.Errorf("Actual: '%d', expected: '%d'", v[0].pathIndex, 1)
			t.Errorf("Actual: '%s', expected: '%s'", v[0].variableName, "mo-ck1")
		}

		if v[1].pathIndex != 3 || v[1].variableName != "m-o-ck2" {
			t.Errorf("Actual: '%d', expected: '%d'", v[1].pathIndex, 3)
			t.Errorf("Actual: '%s', expected: '%s'", v[1].variableName, "m-o-ck2")
		}

		if v[2].pathIndex != 5 || v[2].variableName != "mock3" {
			t.Errorf("Actual: '%d', expected: '%d'", v[2].pathIndex, 5)
			t.Errorf("Actual: '%s', expected: '%s'", v[2].variableName, "mock3")
		}
	}
}

func TestExtractPathVariableNames_when_empty(t *testing.T) {
	// GIVEN
	var path string = "/"

	// WHEN
	v := extractPathVariableNames(path)

	// THEN
	if v != nil {
		t.Errorf("Actual: '%v', expected: '%v'", v, nil)
	}
}

func TestExtractPathVariableValues_when_nominal(t *testing.T) {
	// GIVEN
	var path string = "/a/111111/bbb/222222/a-b-c1/333333"

	pathVariables := []PathVariable{
		PathVariable{pathIndex: 1, variableName: "mo-ck1"},
		PathVariable{pathIndex: 3, variableName: "m-o-ck2"},
		PathVariable{pathIndex: 5, variableName: "mock3"},
	}

	// WHEN
	actual := extractPathVariableValues(path, pathVariables)

	// THEN
	if len(actual) != 3 {
		t.Errorf("Actual: '%d', expected: '%d'", len(actual), 3)
	}

	if actual["mo-ck1"] != "111111" {
		t.Errorf("Actual: '%s', expected: '%s'", actual["mo-ck1"], "111111")
	}

	if actual["m-o-ck2"] != "222222" {
		t.Errorf("Actual: '%s', expected: '%s'", actual["m-o-ck2"], "222222")
	}

	if actual["mock3"] != "333333" {
		t.Errorf("Actual: '%s', expected: '%s'", actual["mock3"], "333333")
	}
}

func TestExtractPathVariableValues_when_empty(t *testing.T) {
	// GIVEN
	var path string = "/a/111111/bbb/222222/a-b-c1/333333"

	var pathVariables []PathVariable = nil

	// WHEN
	actual := extractPathVariableValues(path, pathVariables)

	// THEN
	if actual == nil {
		t.Errorf("Expected: '%v'", nil)
	}

	if len(actual) != 0 {
		t.Errorf("Actual: '%d', expected: '%d'", len(actual), 0)
	}
}

func TestToRegexPath_when_nominal(t *testing.T) {
	// GIVEN
	var path string = "/a/{mo-ck1}/bbb/{m-o-ck2}/a-b-c1/{mock3}"

	// WHEN
	regex := toRegexPath(path)

	// THEN
	s := "[a-zA-Z0-9_-]+"
	expected := fmt.Sprintf("/a/%s/bbb/%s/a-b-c1/%s", s, s, s)
	if regex.String() != expected {
		t.Errorf("Actual: '%s', expected: '%s'", regex.String(), expected)
	}
}