package command

type FieldSeparator string
type OutputFieldSeparator string

type Variable struct {
	Name  string
	Value any
}

type flags struct {
	FieldSeparator       FieldSeparator
	OutputFieldSeparator OutputFieldSeparator
	Variables            map[string]any
}

func (f FieldSeparator) Configure(flags *flags)       { flags.FieldSeparator = f }
func (o OutputFieldSeparator) Configure(flags *flags) { flags.OutputFieldSeparator = o }
func (v Variable) Configure(flags *flags) {
	if flags.Variables == nil {
		flags.Variables = make(map[string]any)
	}
	flags.Variables[v.Name] = v.Value
}
