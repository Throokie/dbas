---
name: code-review
version: 1.0.0
description: Code review best practices and checklist skill
---

## Code Review Instructions

When conducting code reviews, follow this checklist:

1. Does the code solve the right problem? Check against requirements
2. Is the code maintainable? Check naming, structure, comments
3. Are there tests? Check coverage of new/modified code
4. Is error handling correct? Check all error paths are handled
5. Are there security issues? Check for injection, XSS, auth bypass
6. Is performance acceptable? Check N+1 queries, unnecessary allocations
7. Is the change backward compatible? Check API and DB schema changes
8. Are dependencies justified? Check for unnecessary new dependencies
9. Is documentation updated? Check README, API docs, inline comments
10. Are there debugging artifacts? Check for commented code, TODOs, console.log
