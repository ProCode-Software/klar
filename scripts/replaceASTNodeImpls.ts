import { appendFileSync } from 'fs'
import { join } from 'path'

const astTypesFile = await Bun.file(
    join(import.meta.dir, '../internal/ast/ast_types.go')
).text()

const outPath = join(import.meta.dir, '../internal/ast/equal_nodes.go')

// TODO: also generate for Node.Walk()
const nodes = astTypesFile.matchAll(/^\s*type ([\w_]+) struct/gm)

Bun.write(outPath, `package ast\n`)
for (const [, typeName] of nodes) {
    appendFileSync(outPath, `\nfunc (${typeName}) Equal(Node) bool { return false }\n`)
}
