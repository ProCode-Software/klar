package typed

import "fmt"

func (d BaseDecl) GetName() string {
	return d.Name
}

func (d *VariableDecl) GetName() string {
	return fmt.Sprint(d.Idents)
}

// Statements
func (d BaseDecl) stmt()      {}
func (d *VariableDecl) stmt() {}
