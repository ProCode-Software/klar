---
trigger: always_on
---

- Always use the latest Go syntax and features, such as with `range` loops, `strings.SplitSeq`, `WaitGroup.Go`, `new(expr)`. I am using Go 1.26
- Do not create or run tests unless I explicitly ask you to
- Do not generate or remove blank lines between code. No adding or removing comments unless a part of the code really needs explaination (or I ask you to)
- Variables of 3 or more should be grouped by `var ( ... )` in Go.
- Use (\*testing.B).Loop() instead of b.N
- Use `min(...)` and `max(...)` when appropriate
- Use `cmp.Or()` when appropriate and possible
- Up to 90 columns of code. If some parentheses exceed this, it is ok. Try to go no more than 94. Doc comment should be below 90, and up to 95 if the comment is only one line.
