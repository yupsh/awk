package command

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	gloo "github.com/gloo-foo/framework"
)

// Context provides access to awk's execution context for each line
type Context struct {
	// Fields contains the split fields from the current line
	// Fields[0] is $0 (the whole line)
	// Fields[1] is $1 (first field), etc.
	Fields []string

	// NR is the current record (line) number (1-based)
	NR int64

	// NF is the number of fields in the current record
	NF int

	// FS is the input field separator
	FS string

	// OFS is the output field separator (used when printing multiple fields)
	OFS string

	// Variables allows access to user-defined variables
	Variables map[string]any

	// RS is the record separator (usually newline)
	RS string
}

// Field returns the field at the given index (0 = whole line, 1 = first field, etc.)
func (c *Context) Field(index int) string {
	if index < 0 || index >= len(c.Fields) {
		return ""
	}
	return c.Fields[index]
}

// SetField sets the value of a field
func (c *Context) SetField(index int, value string) {
	if index < 0 {
		return
	}
	// Expand fields if necessary
	for len(c.Fields) <= index {
		c.Fields = append(c.Fields, "")
	}
	c.Fields[index] = value
	c.NF = len(c.Fields) - 1 // Don't count $0
}

// Var returns a variable value
func (c *Context) Var(name string) any {
	if c.Variables == nil {
		return nil
	}
	return c.Variables[name]
}

// SetVar sets a variable value
func (c *Context) SetVar(name string, value any) {
	if c.Variables == nil {
		c.Variables = make(map[string]any)
	}
	c.Variables[name] = value
}

// Print formats and returns a string with fields separated by OFS
func (c *Context) Print(values ...any) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, c.OFS)
}

// Program defines the interface for awk-style programs
// All methods are optional - implement only what you need
type Program interface {
	// Begin is called once before processing any lines
	// Use this for initialization
	Begin(ctx *Context) error

	// Condition is called for each line to determine if Action should run
	// Return true to run the action, false to skip
	Condition(ctx *Context) bool

	// Action is called for each line where Condition returns true
	// Return the output string and whether to emit it
	Action(ctx *Context) (output string, emit bool)

	// End is called once after processing all lines
	// Return any final output and an error if needed
	End(ctx *Context) (output string, err error)
}

// SimpleProgram provides default implementations for all Program methods
// Embed this in your program struct and override only what you need
type SimpleProgram struct{}

func (SimpleProgram) Begin(ctx *Context) error              { return nil }
func (SimpleProgram) Condition(ctx *Context) bool           { return true }
func (SimpleProgram) Action(ctx *Context) (string, bool)    { return ctx.Field(0), true }
func (SimpleProgram) End(ctx *Context) (string, error)      { return "", nil }

type command struct {
	program Program
	inputs  gloo.Inputs[gloo.File, flags]
}

func Awk(program Program, parameters ...any) gloo.Command {
	cmd := command{
		program: program,
		inputs:  gloo.Initialize[gloo.File, flags](parameters...),
	}
	if cmd.inputs.Flags.FieldSeparator == "" {
		cmd.inputs.Flags.FieldSeparator = " "
	}
	if cmd.inputs.Flags.OutputFieldSeparator == "" {
		cmd.inputs.Flags.OutputFieldSeparator = " "
	}
	return cmd
}

func (c command) Executor() gloo.CommandExecutor {
	return c.inputs.Wrap(func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
		// Initialize context
		awkCtx := &Context{
			NR:        0,
			FS:        string(c.inputs.Flags.FieldSeparator),
			OFS:       string(c.inputs.Flags.OutputFieldSeparator),
			RS:        "\n",
			Variables: make(map[string]any),
		}

		// Copy initial variables from flags
		for k, v := range c.inputs.Flags.Variables {
			awkCtx.Variables[k] = v
		}

		// Call Begin
		if err := c.program.Begin(awkCtx); err != nil {
			return fmt.Errorf("BEGIN: %w", err)
		}

		// Process lines
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			awkCtx.NR++
			line := scanner.Text()

		// Split into fields
		awkCtx.Fields = make([]string, 0, 16)
		awkCtx.Fields = append(awkCtx.Fields, line) // $0

		var fields []string
		if awkCtx.FS == " " {
			// Default: split on whitespace
			fields = strings.Fields(line)
		} else {
			// Custom separator
			if line == "" {
				// Empty line has no fields, regardless of separator
				fields = []string{}
			} else {
				fields = strings.Split(line, awkCtx.FS)
			}
		}
		awkCtx.Fields = append(awkCtx.Fields, fields...)
		awkCtx.NF = len(fields)

			// Check condition
			if !c.program.Condition(awkCtx) {
				continue
			}

			// Execute action
			output, emit := c.program.Action(awkCtx)
			if emit {
				fmt.Fprintln(stdout, output)
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}

		// Call End
		endOutput, err := c.program.End(awkCtx)
		if err != nil {
			return fmt.Errorf("END: %w", err)
		}
		if endOutput != "" {
			fmt.Fprintln(stdout, endOutput)
		}

		return nil
	})
}
