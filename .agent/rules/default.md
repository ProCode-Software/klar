---
trigger: always_on
---

- Always use the latest Go syntax and features, such as with `range` loops, `strings.SplitSeq`, `WaitGroup.Go`. I am using Go 1.25
- Do not create or run tests unless I explicitly ask you to
- Do not generate or remove blank lines between code. No adding or removing comments unless a part of the code really needs explaination
- Variables of 3 or more should be grouped by `var ( ... )` in Go.
- Use (*testing.B).Loop() instead of b.N
