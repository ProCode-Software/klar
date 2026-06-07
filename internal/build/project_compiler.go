package build

type ProjectCompiler struct {
	*Compiler
	Inputs []ProjectInput
}

type Result struct {
	
}

func NewProjectCompiler(c *Compiler) *ProjectCompiler {
	return &ProjectCompiler{Compiler: c}
}


func (pc *ProjectCompiler) Compile() (*Result, error) {
	
}