enum Gender { case male, female, nonBinary, other }

struct Person {
    var name: String
    var age: Int
    var gender: Gender?

    func greet(person otherPerson: String) {
        print("Hello \(otherPerson)! My name is \(name).")
    }
}

var people: [Person] = [
    Person(name: "John", age: 34, gender: .male),
    Person(name: "Jane", age: 32, gender: .female),
]
print(people)
