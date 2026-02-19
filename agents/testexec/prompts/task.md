# Test Executor Task Prompt

## Test Writing Task

**What Was Fixed**: {{.Description}}

**Test File**: {{.TestFile}}
**Test Framework**: {{.Framework}}
**Tests to Write**: {{.TestCount}}

{{if .TestCases}}
**Test Cases**:
{{range $i, $tc := .TestCases}}
{{$i}}. {{$tc}}
{{end}}
{{end}}

{{if .FixCode}}
**Fixed Code**:
```go
{{.FixCode}}
```
{{end}}

## Your Task

Write tests that verify the fix works correctly. Include edge cases and error conditions.

Output the complete test file.
