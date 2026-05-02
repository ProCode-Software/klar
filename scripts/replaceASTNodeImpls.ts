import { appendFileSync } from 'fs'
import { join } from 'path'

const astTypesFile = await Bun.file(
    join(import.meta.dir, '../internal/ast/ast_types.go')
).text()

const equalFile = join(import.meta.dir, '../internal/ast/equal_nodes.go')
const walkFile = join(import.meta.dir, '../internal/ast/walk_nodes.go')

// TODO: also generate for Node.Walk()
const nodes = astTypesFile.matchAll(/^\s*type ([\w_]+) struct/gm)

Bun.write(equalFile, `package ast\n`)
Bun.write(walkFile, `package ast\n`)
for (const [, typeName] of nodes) {
    if (typeName == 'BaseNode') continue
    appendFileSync(equalFile, `\nfunc (${typeName}) Equal(Node) bool { return false }\n`)
    appendFileSync(
        walkFile,
        `\nfunc (${typeName}) Walk(Visitor, *Cursor) StopCode { return 0 }\n`
    )
}
