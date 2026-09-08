class KlarString extends String {
    /** @type {Intl.Segments | null} */
    #cachedSegments = null
    constructor(str) {
        super(str)
    }
    get length() { // Unreachable
        if (this.#cachedSegments == null) this.#makeSegments()
        return this.#cachedSegments.length
    }
    #makeSegments() {
        this.#cachedSegments = new Intl.Segmenter(undefined, {
            granularity: 'grapheme',
        }).segment(this)
    }
    [Symbol.iterator]() {
        if (this.#cachedSegments == null) this.#makeSegments()
        return this.#cachedSegments[Symbol.iterator]().map(s => s.segment)
    }
}

const strings = ['Hello', '👨‍👩‍👧', '🏳️‍🌈']
for (const str of strings) {
    const ks = new KlarString(str)
    console.log(
        'Klar String:',
        ks,
        ks.length,
        ks instanceof String,
        typeof ks,
        ks + ' world',
        [...ks],
    )
    console.log('JS String:', str.length)
    console.log()
}
