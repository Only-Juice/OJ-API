package sandbox

/* runHandler.go
 * JudgeResult: Store judge result such as AC、WA、RE、CE
 * SandboxCompileResult: Compile result when using sandbox judge, execute, calculate score
 * SandboxJudgeResult: Store result when using sandbox judge, execute score
 * SandboxScoreResult: Store result when using sandbox calculate score
 * SandboxResult: Store SandboxJudgeResult
 * CompileTask: Task for exe file and include test.
 * CompileFile: Total file for every task.
 */
type JudgeResult string

const (
	SYSTEM_FAILED         JudgeResult = "SYSTEM_ERROR"
	WAITING_TO_JUDGE      JudgeResult = "WAITING_TO_JUDGE"
	JUDGING               JudgeResult = "JUDGING"
	ACCEPTED              JudgeResult = "ACCEPTED"
	WRONG_ANSWER          JudgeResult = "WRONG_ANSWER"
	COMPILE_ERROR         JudgeResult = "COMPILE_ERROR"
	RUNTIME_ERROR         JudgeResult = "RUNTIME_ERROR"
	TIME_LIMIT_EXCEEDED   JudgeResult = "TIME_LIMIT_EXCEEDED"
	MEMORY_LIMIT_EXCEEDED JudgeResult = "MEMORY_LIMIT_EXCEEDED"
)

type SandboxJudgeResult struct {
	Target string `json:"target"`
	Status string `json:"status"`
	Result string `json:"result"`
}

type SandboxScoreResult struct {
	Target string  `json:"target"`
	Status string  `json:"status"`
	Score  float64 `json:"score"`
	Result string  `json:"result"`
}

type SandboxResult struct {
	CompileResult    []SandboxJudgeResult `json:"compileresult"`
	ExecuteResult    []SandboxJudgeResult `json:"executeresult"`
	JudgeScoreResult []SandboxScoreResult `json:"judgescoreresult"`
}

type CompileTask struct {
	Target string   `json:"target"`
	Suite  []string `json:"suite"`
}

type CompileFile struct {
	Task []CompileTask `json:"task"`
}

/* result.go */
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
