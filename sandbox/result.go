package sandbox

import (
	"encoding/json"
	"time"
)

// === 結構定義 ===

type Failure struct {
	Failure string `json:"failure"`
	Type    string `json:"type"`
}

type TestCase struct {
	Name      string    `json:"name"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Status    string    `json:"status"`
	Result    string    `json:"result"`
	Timestamp string    `json:"timestamp"`
	Time      string    `json:"time"`
	Classname string    `json:"classname"`
	Failures  []Failure `json:"failures,omitempty"`
}

type TestSuite struct {
	Name      string     `json:"name"`
	MaxScore  int        `json:"maxscore"`
	GetScore  int        `json:"getscore"`
	Tests     int        `json:"tests"`
	Failures  int        `json:"failures"`
	Disabled  int        `json:"disabled"`
	Errors    int        `json:"errors"`
	Timestamp string     `json:"timestamp"`
	Time      string     `json:"time"`
	TestSuite []TestCase `json:"testsuite"`
}

type AllTests struct {
	Tests      int         `json:"tests"`
	Failures   int         `json:"failures"`
	Disabled   int         `json:"disabled"`
	Errors     int         `json:"errors"`
	Timestamp  string      `json:"timestamp"`
	Time       string      `json:"time"`
	Name       string      `json:"name"`
	TestSuites []TestSuite `json:"testsuites"`
}

// Generate System Message when error occur
// Format reference by gTest JSON file
func NewErrorResult(errType, errMsg string) string {
	now := time.Now().UTC().Format(time.RFC3339)

	failure := Failure{
		Failure: errMsg,
		Type:    "",
	}

	testCase := TestCase{
		Name:      errType,
		File:      "System Message",
		Line:      0,
		Status:    "failed",
		Result:    "ERROR",
		Timestamp: now,
		Time:      "0s",
		Classname: "System",
		Failures:  []Failure{failure},
	}

	testSuite := TestSuite{
		Name:      "ERROR_SUITE",
		MaxScore:  100,
		GetScore:  0,
		Tests:     1,
		Failures:  1,
		Disabled:  0,
		Errors:    0,
		Timestamp: now,
		Time:      "0s",
		TestSuite: []TestCase{testCase},
	}

	all := AllTests{
		Tests:      1,
		Failures:   1,
		Disabled:   0,
		Errors:     0,
		Timestamp:  now,
		Time:       "0s",
		Name:       "AllTests",
		TestSuites: []TestSuite{testSuite},
	}

	jsonBytes, _ := json.MarshalIndent(all, "", "  ")
	return string(jsonBytes)
}
