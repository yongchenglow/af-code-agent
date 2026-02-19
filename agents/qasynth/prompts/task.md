# QA Synthesizer Task Prompt

## QA Synthesis Task

**Issue**: %s
**Description**: %s

## Validation Results
- Syntax Valid: %v
- Linting Passed: %v
- Tests Passed: %v

%s

## Decision Criteria

Make a decision based on:
1. **APPROVE** if all validations pass and the fix is complete
2. **FIX** if validations fail or the fix is incomplete
3. **BLOCK** if there are critical issues that cannot be fixed automatically

Output your decision as JSON:
```json
{
  "action": "APPROVE|FIX|BLOCK",
  "summary": "...",
  "feedback": ["..."],
  "stuck": false
}
```
