enum Gender { case male, female, nonBinary, other }

struct Person {
    var name: String
    var age: Int
    var gender: Gender?

    func greet(_ otherPerson: String) {
        print("Hello \(otherPerson)! My name is \(self.name).")
    }
}

var people: [Person] = [
    Person(name: "John", age: 34, gender: .male),
    Person(name: "Jane", age: 32, gender: .female),
]
print(people)
people[0].greet("Lucy")
