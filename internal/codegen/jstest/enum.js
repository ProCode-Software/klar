/**
 * This file is for evaluating possible JS outputs for Klar enums. Please send
 * feedback in GitHub discussions.
 */

class TokenType {
    static #names = ['leftParen', 'string']
    rawValue
    constructor(rawValue) {
        this.rawValue = rawValue
    }
    get name() {
        return TokenType.#names[this.rawValue]
    }
    equals(other) {
        // TODO: Check equality of parameters
        return this.rawValue === other.rawValue
    }
    static from(rawValue) {
        if (!this.#names[rawValue]) return undefined
        return new TokenType(rawValue)
    }

    // Variants
    static leftParen = new TokenType(0)
    static string(content) {
        const $ = new TokenType(1)
        $.content = content
        return $
    }
}

const lp = TokenType.leftParen
const str = TokenType.string('Hello')
console.log(lp, lp.name, lp.rawValue)
console.log(str, str.name, str.rawValue, str.content)
console.log(lp instanceof TokenType, str instanceof TokenType)
console.log(TokenType.from(0))
console.log(lp.equals(TokenType.from(0))) // Expected: true
