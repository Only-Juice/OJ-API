package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var boxPathRegex = regexp.MustCompile(`(?m)/[^:\n]*/box/`)

// TestSuite represents a test suite structure from the input JSON
type TestSuite struct {
	Name      string     `json:"name"`
	MaxScore  int        `json:"maxscore"`
	GetScore  float64    `json:"getscore"`
	Tests     int        `json:"tests"`
	Failures  int        `json:"failures"`
	Disabled  int        `json:"disabled"`
	Errors    int        `json:"errors"`
	Timestamp string     `json:"timestamp"`
	Time      string     `json:"time"`
	TestSuite []TestCase `json:"testsuite"`
}

// TestCase represents individual test case
type TestCase struct {
	Name      string    `json:"name"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Status    string    `json:"status"`
	Result    string    `json:"result"`
	Timestamp string    `json:"timestamp"`
	Time      string    `json:"time"`
	ClassName string    `json:"classname"`
	Failures  []Failure `json:"failures,omitempty"`
	Errors    []Error   `json:"errors,omitempty"`
}

// Failure represents a test failure
type Failure struct {
	Failure string `json:"failure"`
	Type    string `json:"type"`
}

// Error represents a test error
type Error struct {
	Error string `json:"error"`
	Type  string `json:"type"`
}

// InputJSON represents the structure of the input JSON file
type InputJSON struct {
	Tests      int         `json:"tests"`
	Failures   int         `json:"failures"`
	Disabled   int         `json:"disabled"`
	Errors     int         `json:"errors"`
	Timestamp  string      `json:"timestamp"`
	Time       string      `json:"time"`
	Name       string      `json:"name"`
	TestSuites []TestSuite `json:"testsuites"`
}

// ScoreTestSuite represents test suite scoring structure
type ScoreTestSuite struct {
	TestSuite string `json:"testsuite"`
	Score     int    `json:"score"`
}

// ScoreJSON represents the structure of the score JSON file
type ScoreJSON struct {
	HomeworkName string           `json:"homework_name"`
	Semester     string           `json:"semester"`
	TestSuites   []ScoreTestSuite `json:"testsuites"`
}

// JSONParser handles JSON parsing and score calculation
type JSONParser struct {
	parsePath string
	scorePath string
	scoreFile ScoreJSON
	inputFile InputJSON
	score     float64
	task      map[string]int
}

// NewJSONParser creates a new JSONParser instance
func NewJSONParser(parsePath, scorePath string) (*JSONParser, error) {
	parser := &JSONParser{
		parsePath: parsePath,
		scorePath: scorePath,
		task:      make(map[string]int),
	}

	// Read and parse input JSON file
	inputData, err := os.ReadFile(parsePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %v", err)
	}

	err = json.Unmarshal(inputData, &parser.inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input JSON: %v", err)
	}

	// Read and parse score JSON file
	scoreData, err := os.ReadFile(scorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read score file: %v", err)
	}

	err = json.Unmarshal(scoreData, &parser.scoreFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse score JSON: %v", err)
	}

	parser.parseScore()
	return parser, nil
}

// parseScore parses the score configuration
func (jp *JSONParser) parseScore() {
	for _, testSuite := range jp.scoreFile.TestSuites {
		jp.task[testSuite.TestSuite] = testSuite.Score
	}
}

// Parse calculates the score based on test results
func (jp *JSONParser) Parse() {
	for i := range jp.inputFile.TestSuites {
		name := jp.inputFile.TestSuites[i].Name
		ac := 0.0 // accepted count
		wa := 0.0 // wrong answer count

		for _, suite := range jp.inputFile.TestSuites[i].TestSuite {
			// Check if individual test case has failures or errors
			if len(suite.Failures) > 0 {
				for j := range suite.Failures {
					suite.Failures[j].Failure = boxPathRegex.ReplaceAllString(suite.Failures[j].Failure, "")
				}
			}
			if len(suite.Failures) > 0 || len(suite.Errors) > 0 {
				wa += 1.0
			} else {
				ac += 1.0
			}
		}

		if maxScore, exists := jp.task[name]; exists && (ac+wa) > 0 {
			jp.inputFile.TestSuites[i].MaxScore = maxScore
			jp.inputFile.TestSuites[i].GetScore = (ac / (ac + wa)) * float64(maxScore)
			jp.score += (ac / (ac + wa)) * float64(maxScore)
		}
	}
}

// GetScore returns the calculated score
func (jp *JSONParser) GetScore() float64 {
	return jp.score
}

// writeScore writes the score to score.txt file, keeping the maximum score
func writeScore(score float64) (bool, error) {
	const file = "score.txt"
	oldScore := -1.0
	higher := true

	// Check if file exists and read old score
	if _, err := os.Stat(file); err == nil {
		data, err := os.ReadFile(file)
		if err == nil {
			line := strings.TrimSpace(string(data))
			if line != "" {
				if parsed, err := strconv.ParseFloat(line, 64); err == nil {
					oldScore = parsed
				}
			}
		}
	}

	// Keep the maximum score
	newScore := score
	if oldScore > score {
		newScore = oldScore
		higher = false
	}

	return higher, os.WriteFile(file, []byte(fmt.Sprintf("%.2f\n", newScore)), 0644)
}

// writeJSONFile copies the JSON file content to message.txt
func (jp *JSONParser) writeJSONFile(jsonPath string) error {
	outputFile, err := os.Create("message.txt")
	if err != nil {
		return fmt.Errorf("failed to create output file message.txt: %v", err)
	}
	defer outputFile.Close()

	newJSONfile, err := json.MarshalIndent(jp.inputFile, " ", " ")
	if err != nil {
		return fmt.Errorf("failed to parse json file: %v", err)
	}

	if _, err := outputFile.Write(newJSONfile); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Copied content from %s to message.txt\n", jsonPath)
	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <gtest.json> <score.json>\n", os.Args[0])
		os.Exit(1)
	}

	parser, err := NewJSONParser(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating parser: %v\n", err)
		os.Exit(1)
	}

	parser.Parse()
	score := parser.GetScore()
	fmt.Printf("%.2f\n", score)

	higher, err := writeScore(score)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing score: %v\n", err)
		os.Exit(1)
	}

	if higher {
		if err := parser.writeJSONFile(os.Args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing JSON file: %v\n", err)
			os.Exit(1)
		}
	}
}
