import { $ } from 'bun'
import { join } from 'path'
import { styleText } from 'util'

// This script uploads and updates RFCs from the Markdown file as individual
// GitHub discussions.
//
// GitHub CLI (gh) is required to run this script. You'll need to be signed in
// with permissions to post in the RFCs category.

const rfcContent = await Bun.file(join(import.meta.dirname, '../docs/RFCs.md')).text()
const items = rfcContent
    .split(/\n(?=\d+\.\s+)/g)
    .filter(item => /^\d+\.\s+/.test(item.trim()))

const bodies = new Map<number, string>()
const discussions = await loadDiscussions()
const addedOrCreated = new Map<string, true>()

for (let item of items) {
    const [match, i2] = (item = item.trim()).match(/^(\d+)\.\s+/)!
    let i = +i2!

    item = item
        .substring(match.length) // Cut line number
        .replaceAll(/^(\t| {4})/gm, '') // Dedent

    if (!item.startsWith('**')) throw new Error(`Item isn't bolded: #${i}`)
    const [match2, title] = item.match(/^\*\*(.+?)(?::|\?)?\*\*\s*(?:|-|\n)\s*/)!
    let body = item.substring(match2.length)
    // Strip the hyphen in "**Title** - Body", but not in a list item on the next line
    if (body.startsWith(' - ')) body = body.substring(3)
    body = body.trim()
    bodies.set(+i, body)
    await updateOrCreateDiscussion(+i, title!.trim(), body)
}

// List RFC discussions that are no longer in RFCs.md
staleDiscussions()

async function loadDiscussions(): Promise<{ number: number; title: string }[]> {
    const { discussions } =
        await $`gh discussion list -R 'ProCode-Software/klar' --json number,title`
            .quiet()
            .json()
    return discussions
}

function staleDiscussions() {
    const rfcRegex = /^RFC #\d+:\s*/
    for (const { title } of discussions) {
        // There is a discussion in the RFCs category called "Klar RFCs".
        // Don't care about it
        if (!rfcRegex.test(title)) continue

        const actualTitle = title.replace(rfcRegex, '').trim()
        if (!addedOrCreated.has(actualTitle)) {
            console.log(`${styleText('yellow', 'Stale')} discussion: ${title}`)
        }
    }
}

async function updateOrCreateDiscussion(rfcNum: number, title: string, body: string) {
    const disc = discussions.find(({ title: discTitle }) => discTitle.includes(title))
    if (!disc) {
        console.log(
            `${styleText('green', 'Creating')} discussion for RFC #${rfcNum}: ${title}`
        )
        await createDiscussion(rfcNum, title, body)
        return
    }
    const { number: discNum } = disc
    await $`gh discussion edit -R 'ProCode-Software/klar' ${discNum} \
        -t 'RFC #${rfcNum}: ${title}' -b '${body}'`.quiet()
    // If the content remains the same, GitHub won't show the discussion as updated,
    // so we don't have to worry about re-editing the same content
    console.log(`${styleText('blue', 'Synced')} ${title}`)
    addedOrCreated.set(title, true)
}

async function createDiscussion(rfcNum: number, title: string, body: string) {
    await $`gh discussion create -R 'ProCode-Software/klar' -c RFCs \
        -t 'RFC #${rfcNum}: ${title}' -b '${body}'`.quiet()
    addedOrCreated.set(title, true)
}
