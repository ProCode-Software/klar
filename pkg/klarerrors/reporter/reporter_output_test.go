package reporter

const singleLine_expected = `
SyntaxError: Use '=' instead of ':=' to set a default (colonEqual)

  ┌─ main.klar:1:3
1 │ x := 1
  ·   ━━ Use ':=' instead
`

const singleLine1Highlight_expected = `
SyntaxError: Use '=' instead of ':=' to set a default (colonEqual)

  ┌─ main2.klar:1:3
1 │ x := 1
  · ^ ━━ Use ':=' instead
  · ╰─ This is 'x'
`

const singleLine2Highlights_expected = `
SyntaxError: Use '=' instead of ':=' to set a default (colonEqual)

  ┌─ main.klar:1:3
1 │ x := 1
  · ^ ┯━ ^ 'x' is set to 1
  · │ ╰─ Use ':=' instead
  · ╰─ This is 'x'
`

const Multiline2Highlights_expected = `
SyntaxError: Braces are required around this statement (requiredBraces)

  ┌─ main.klar:2:3
1 │ deleteThisLine()
  · ╭━━━━━━━━━━━━━━━
2 │ │ x := 1
  · │   ┯━ ^ 'x' is set to 1
  · │   ╰─ Something else is better
  · ╰━━━━━━━ Please delete this
`
