# Planner Task Prompt

## Code Review Task

{{.PRInfo}}

## Files to Review

{{.FilesInfo}}

## Review Instructions

Analyze the code changes above and identify:
1. **Bugs**: Logic errors, null dereferences, type mismatches, edge cases
2. **Security**: SQL injection, XSS, authentication flaws, secrets, input validation
3. **Performance**: N+1 queries, inefficient algorithms, memory leaks
4. **Maintainability**: Complex code, poor naming, missing tests

For each issue, provide:
- Exact file path and line number
- Clear description of the problem
- Concrete fix suggestion
- Severity level (Critical/High/Medium/Low)

Output your review as a JSON object with:
```json
{
  "issues": [...],
  "summary": "...",
  "recommendation": "APPROVE|REQUEST_CHANGES"
}
```
