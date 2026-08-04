package api

type TestCase struct {
	Stdin          string `json:"stdin"`
	ExpectedStdout string `json:"expected_stdout"`
}

type ExecuteRequest struct {
	Language       string     `json:"language"`
	Code           string     `json:"code"`
	FileBase64     string     `json:"file_base64"`
	Filename       string     `json:"filename"`
	Stdin          string     `json:"stdin"`
	ExpectedStdout string     `json:"expected_stdout"`
	TestCases      []TestCase `json:"test_cases"`

	MemoryMB      int64  `json:"memory_mb"`
	CPU           string `json:"cpu"`
	TimeoutSec    int    `json:"timeout_sec"`
	TmpLimitMB    int    `json:"tmp_limit_mb"`
	MaxFileSizeMB int    `json:"max_file_size_mb"`
	MaxOpenFiles  int    `json:"max_open_files"`
}

type GraderExecution struct {
	Stdout           string `json:"stdout"`
	Stderr           string `json:"stderr"`
	DurationMs       int64  `json:"duration_ms"`
	CpuUserTimeUs    int64  `json:"cpu_user_time_us"`
	CpuSystemTimeUs  int64  `json:"cpu_system_time_us"`
	MemoryPeak       int64  `json:"memory_peak_bytes"`
	ExitCode         int    `json:"exit_code"`
	Status           string `json:"status"`
	LimitMemoryBytes int64  `json:"limit_memory_bytes"`
	LimitCPU         string `json:"limit_cpu"`
	LimitTimeoutSec  int    `json:"limit_timeout_sec"`
	LimitFileSizeMax int64  `json:"limit_file_size_bytes"`
	LimitOpenFiles   int    `json:"limit_open_files"`
}

type ExecuteResponse struct {
	Mode            string           `json:"mode"`
	Result          string           `json:"result"`
	ErrorType       string           `json:"error_type,omitempty"`
	Expected        string           `json:"expected,omitempty"`
	Actual          string           `json:"actual,omitempty"`
	FailedTestIndex *int             `json:"failed_test_index,omitempty"`
	PassedCount     int              `json:"passed_count"`
	TotalCount      int              `json:"total_count"`
	Execution       *GraderExecution `json:"execution,omitempty"`
}
