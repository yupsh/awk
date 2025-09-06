package opt

// Custom types for parameters
type Program string
type ScriptFile string
type FieldSeparator string
type Variable map[string]string

// Flags represents the configuration options for the awk command
type Flags struct {
	Program        Program        // AWK program to execute
	ScriptFile     ScriptFile     // File containing AWK script
	FieldSeparator FieldSeparator // Field separator
	Variables      Variable       // Variable assignments
}

// Configure methods for the opt system
func (p Program) Configure(flags *Flags)        { flags.Program = p }
func (s ScriptFile) Configure(flags *Flags)     { flags.ScriptFile = s }
func (f FieldSeparator) Configure(flags *Flags) { flags.FieldSeparator = f }
func (v Variable) Configure(flags *Flags) {
	if flags.Variables == nil {
		flags.Variables = make(map[string]string)
	}
	for k, val := range v {
		flags.Variables[k] = val
	}
}
