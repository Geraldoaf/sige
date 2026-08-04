package api

import (
	"sige/internal/sandbox"
	"strings"
	"sync"
)

func runInterpreter(runSandbox func(string) (sandbox.ExecutionResult, error), stdin string) ExecuteResponse {
	res, execErr := runSandbox(stdin)

	var response ExecuteResponse
	response.Mode = "interpreter"
	response.TotalCount = 1

	response.Execution = &GraderExecution{
		Stdout:           res.Stdout,
		Stderr:           res.Stderr,
		DurationMs:       res.Duration.Milliseconds(),
		CpuUserTimeUs:    res.CPUUserTime.Microseconds(),
		CpuSystemTimeUs:  res.CPUSystemTime.Microseconds(),
		MemoryPeak:       res.MemoryPeak,
		ExitCode:         res.ExitCode,
		Status:           res.Status,
		LimitMemoryBytes: res.LimitMemoryBytes,
		LimitCPU:         res.LimitCPU,
		LimitTimeoutSec:  res.LimitTimeoutSec,
		LimitFileSizeMax: res.LimitFileSizeMax,
		LimitOpenFiles:   res.LimitOpenFiles,
	}

	if execErr != nil || res.Status != "success" {
		response.Result = "failed"
		response.ErrorType = res.Status
		response.PassedCount = 0
	} else {
		response.Result = "completed"
		response.PassedCount = 1
	}

	return response
}

func runSingleEvaluation(runSandbox func(string) (sandbox.ExecutionResult, error), stdin, expectedStdout string) ExecuteResponse {
	res, execErr := runSandbox(stdin)

	var response ExecuteResponse
	response.Mode = "single_evaluation"
	response.TotalCount = 1

	failed := false
	var errorType string

	if execErr != nil || res.Status != "success" {
		failed = true
		errorType = res.Status
	} else {
		actualOut := strings.TrimSpace(res.Stdout)
		expectedOut := strings.TrimSpace(expectedStdout)
		if actualOut != expectedOut {
			failed = true
			errorType = "output_mismatch"
		}
	}

	response.Execution = &GraderExecution{
		Stdout:           res.Stdout,
		Stderr:           res.Stderr,
		DurationMs:       res.Duration.Milliseconds(),
		CpuUserTimeUs:    res.CPUUserTime.Microseconds(),
		CpuSystemTimeUs:  res.CPUSystemTime.Microseconds(),
		MemoryPeak:       res.MemoryPeak,
		ExitCode:         res.ExitCode,
		Status:           res.Status,
		LimitMemoryBytes: res.LimitMemoryBytes,
		LimitCPU:         res.LimitCPU,
		LimitTimeoutSec:  res.LimitTimeoutSec,
		LimitFileSizeMax: res.LimitFileSizeMax,
		LimitOpenFiles:   res.LimitOpenFiles,
	}

	if failed {
		response.Result = "FAIL"
		response.ErrorType = errorType
		response.Expected = expectedStdout
		response.Actual = res.Stdout
		response.PassedCount = 0
	} else {
		response.Result = "PASS"
		response.PassedCount = 1
	}

	return response
}

type testResult struct {
	index      int
	failed     bool
	errorType  string
	execResult sandbox.ExecutionResult
	tc         TestCase
}

func runMultiEvaluation(runSandbox func(string) (sandbox.ExecutionResult, error), testCases []TestCase) ExecuteResponse {
	var response ExecuteResponse
	response.Mode = "multi_evaluation"
	total := len(testCases)
	response.TotalCount = total

	if total == 0 {
		response.Result = "PASS"
		response.PassedCount = 0
		return response
	}

	results := make([]testResult, total)
	var wg sync.WaitGroup
	wg.Add(total)

	for i, tc := range testCases {
		go func(idx int, testCase TestCase) {
			defer wg.Done()

			res, execErr := runSandbox(testCase.Stdin)

			failed := false
			var errorType string

			if execErr != nil || res.Status != "success" {
				failed = true
				errorType = res.Status
			} else {
				actualOut := strings.TrimSpace(res.Stdout)
				expectedOut := strings.TrimSpace(testCase.ExpectedStdout)
				if actualOut != expectedOut {
					failed = true
					errorType = "output_mismatch"
				}
			}

			results[idx] = testResult{
				index:      idx,
				failed:     failed,
				errorType:  errorType,
				execResult: res,
				tc:         testCase,
			}
		}(i, tc)
	}

	wg.Wait()

	passedCount := 0
	var firstFailure *testResult

	for idx := 0; idx < total; idx++ {
		res := results[idx]
		if res.failed {
			if firstFailure == nil {
				firstFailure = &results[idx]
			}
		} else {
			passedCount++
		}
	}

	if firstFailure != nil {
		failedIdx := firstFailure.index
		response.Result = "FAIL"
		response.ErrorType = firstFailure.errorType
		response.Expected = firstFailure.tc.ExpectedStdout
		response.Actual = firstFailure.execResult.Stdout
		response.FailedTestIndex = &failedIdx
		response.PassedCount = passedCount
		response.Execution = &GraderExecution{
			Stdout:           firstFailure.execResult.Stdout,
			Stderr:           firstFailure.execResult.Stderr,
			DurationMs:       firstFailure.execResult.Duration.Milliseconds(),
			CpuUserTimeUs:    firstFailure.execResult.CPUUserTime.Microseconds(),
			CpuSystemTimeUs:  firstFailure.execResult.CPUSystemTime.Microseconds(),
			MemoryPeak:       firstFailure.execResult.MemoryPeak,
			ExitCode:         firstFailure.execResult.ExitCode,
			Status:           firstFailure.execResult.Status,
			LimitMemoryBytes: firstFailure.execResult.LimitMemoryBytes,
			LimitCPU:         firstFailure.execResult.LimitCPU,
			LimitTimeoutSec:  firstFailure.execResult.LimitTimeoutSec,
			LimitFileSizeMax: firstFailure.execResult.LimitFileSizeMax,
			LimitOpenFiles:   firstFailure.execResult.LimitOpenFiles,
		}
		return response
	}

	response.Result = "PASS"
	response.PassedCount = passedCount
	return response
}
