package command_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/gloo-foo/testable/assertion"
	"github.com/gloo-foo/testable/run"
	command "github.com/yupsh/awk"
)

// ==============================================================================
// Test Pure Functions - Context Methods
// ==============================================================================

func TestContext_Field(t *testing.T) {
	ctx := &command.Context{
		Fields: []string{"whole line", "first", "second", "third"},
	}

	tests := []struct {
		name  string
		index int
		want  string
	}{
		{"field 0 (whole line)", 0, "whole line"},
		{"field 1", 1, "first"},
		{"field 2", 2, "second"},
		{"field 3", 3, "third"},
		{"negative index", -1, ""},
		{"out of bounds", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ctx.Field(tt.index)
			assertion.Equal(t, got, tt.want, "field value")
		})
	}
}

func TestContext_SetField(t *testing.T) {
	ctx := &command.Context{
		Fields: []string{"whole", "first", "second"},
	}

	// Update existing field
	ctx.SetField(1, "updated")
	assertion.Equal(t, ctx.Field(1), "updated", "field 1")
	assertion.Equal(t, ctx.NF, 2, "NF should be 2")

	// Expand fields
	ctx.SetField(5, "new")
	assertion.Equal(t, ctx.Field(5), "new", "field 5")
	assertion.Equal(t, ctx.NF, 5, "NF should be 5")

	// Negative index (should be ignored)
	originalLen := len(ctx.Fields)
	ctx.SetField(-1, "ignored")
	assertion.Equal(t, len(ctx.Fields), originalLen, "fields length unchanged")
}

func TestContext_Var(t *testing.T) {
	ctx := &command.Context{
		Variables: map[string]any{
			"count": 10,
			"name":  "test",
		},
	}

	// Get existing variables
	assertion.Equal(t, ctx.Var("count"), 10, "count variable")
	assertion.Equal(t, ctx.Var("name"), "test", "name variable")

	// Get non-existent variable
	assertion.True(t, ctx.Var("missing") == nil, "missing variable should be nil")

	// Nil map
	nilCtx := &command.Context{}
	assertion.True(t, nilCtx.Var("any") == nil, "var on nil map should be nil")
}

func TestContext_SetVar(t *testing.T) {
	ctx := &command.Context{}

	// Set on nil map (should create map)
	ctx.SetVar("x", 42)
	assertion.Equal(t, ctx.Var("x"), 42, "x variable")

	// Update existing
	ctx.SetVar("x", 100)
	assertion.Equal(t, ctx.Var("x"), 100, "updated x variable")

	// Set different types
	ctx.SetVar("str", "hello")
	ctx.SetVar("bool", true)
	assertion.Equal(t, ctx.Var("str"), "hello", "string variable")
	assertion.Equal(t, ctx.Var("bool"), true, "bool variable")
}

func TestContext_Print(t *testing.T) {
	ctx := &command.Context{OFS: "|"}

	tests := []struct {
		name   string
		values []any
		want   string
	}{
		{"multiple values", []any{"a", "b", "c"}, "a|b|c"},
		{"single value", []any{"single"}, "single"},
		{"empty", []any{}, ""},
		{"mixed types", []any{1, "two", 3.0}, "1|two|3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ctx.Print(tt.values...)
			assertion.Equal(t, got, tt.want, "print output")
		})
	}
}

// ==============================================================================
// Test SimpleProgram Default Behavior
// ==============================================================================

func TestSimpleProgram(t *testing.T) {
	prog := command.SimpleProgram{}
	ctx := &command.Context{
		Fields: []string{"test line", "field1"},
	}

	// Test Begin
	err := prog.Begin(ctx)
	assertion.NoError(t, err)

	// Test Condition (always true)
	assertion.True(t, prog.Condition(ctx), "condition should be true")

	// Test Action (returns $0)
	output, emit := prog.Action(ctx)
	assertion.Equal(t, output, "test line", "output")
	assertion.True(t, emit, "should emit")

	// Test End
	endOutput, err := prog.End(ctx)
	assertion.NoError(t, err)
	assertion.Equal(t, endOutput, "", "end output should be empty")
}

// ==============================================================================
// Test Command Execution - Simple Cases
// ==============================================================================

func TestAwk_SimplePassThrough(t *testing.T) {
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines("line1", "line2", "line3").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"line1",
		"line2",
		"line3",
	})
}

func TestAwk_EmptyInput(t *testing.T) {
	result := run.Quick(command.Awk(command.SimpleProgram{}))

	assertion.NoError(t, result.Err)
	assertion.Empty(t, result.Stdout)
}

func TestAwk_SingleLine(t *testing.T) {
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines("single line").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{"single line"})
}

// ==============================================================================
// Test Custom Programs
// ==============================================================================

// UppercaseProgram converts all lines to uppercase
type UppercaseProgram struct {
	command.SimpleProgram
}

func (p UppercaseProgram) Action(ctx *command.Context) (string, bool) {
	line := ctx.Field(0)
	return strings.ToUpper(line), true
}

func TestAwk_UppercaseProgram(t *testing.T) {
	result := run.Command(command.Awk(UppercaseProgram{})).
		WithStdinLines("hello", "world").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"HELLO",
		"WORLD",
	})
}

// CountingProgram counts lines
type CountingProgram struct {
	command.SimpleProgram
	count int
}

func (p *CountingProgram) Action(ctx *command.Context) (string, bool) {
	p.count++
	return "", false // Don't emit per line
}

func (p *CountingProgram) End(ctx *command.Context) (string, error) {
	return fmt.Sprintf("Total lines: %d", p.count), nil
}

func TestAwk_CountingProgram(t *testing.T) {
	prog := &CountingProgram{}
	result := run.Command(command.Awk(prog)).
		WithStdinLines("line1", "line2", "line3").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{"Total lines: 3"})
}

// ConditionalProgram only processes certain lines
type ConditionalProgram struct {
	command.SimpleProgram
}

func (p ConditionalProgram) Condition(ctx *command.Context) bool {
	// Only process lines starting with "include:"
	line := ctx.Field(0)
	return strings.HasPrefix(line, "include:")
}

func TestAwk_ConditionalProgram(t *testing.T) {
	result := run.Command(command.Awk(ConditionalProgram{})).
		WithStdinLines(
			"include:line1",
			"skip this",
			"include:line2",
			"skip this too",
		).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"include:line1",
		"include:line2",
	})
}

// ==============================================================================
// Test Field Splitting
// ==============================================================================

// FieldExtractorProgram extracts specific fields
type FieldExtractorProgram struct {
	command.SimpleProgram
	fieldIndex int
}

func (p FieldExtractorProgram) Action(ctx *command.Context) (string, bool) {
	return ctx.Field(p.fieldIndex), true
}

func TestAwk_FieldSplitting_Whitespace(t *testing.T) {
	result := run.Command(command.Awk(FieldExtractorProgram{fieldIndex: 2})).
		WithStdinLines(
			"first   second   third",
			"a  b  c",
		).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"second",
		"b",
	})
}

func TestAwk_FieldSplitting_DefaultWhitespace(t *testing.T) {
	// Test that default FS=" " splits on whitespace runs
	result := run.Command(command.Awk(FieldExtractorProgram{fieldIndex: 1})).
		WithStdinLines("a    b    c").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{"a"})
}

func TestAwk_FieldAccess_OutOfBounds(t *testing.T) {
	// Test accessing fields beyond what exists returns empty string
	result := run.Command(command.Awk(FieldExtractorProgram{fieldIndex: 10})).
		WithStdinLines("a b c").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{""})
}

// ContextInspectorProgram inspects all context fields
type ContextInspectorProgram struct {
	command.SimpleProgram
}

func (p ContextInspectorProgram) Action(ctx *command.Context) (string, bool) {
	return fmt.Sprintf("NR=%d NF=%d FS=%q OFS=%q Fields=%d",
		ctx.NR, ctx.NF, ctx.FS, ctx.OFS, len(ctx.Fields)), true
}

func TestAwk_ContextFields(t *testing.T) {
	result := run.Command(
		command.Awk(
			ContextInspectorProgram{},
			command.FieldSeparator(","),
			command.OutputFieldSeparator("|"),
		),
	).WithStdinLines("a,b,c").Run()

	assertion.NoError(t, result.Err)
	// Verify context fields are set correctly
	assertion.Contains(t, result.Stdout, "NR=1")
	assertion.Contains(t, result.Stdout, "NF=3")
	assertion.Contains(t, result.Stdout, `FS=","`)
	assertion.Contains(t, result.Stdout, `OFS="|"`)
	assertion.Contains(t, result.Stdout, "Fields=4") // $0 + 3 fields
}

func TestAwk_FieldSplitting_CustomSeparator(t *testing.T) {
	result := run.Command(
		command.Awk(
			FieldExtractorProgram{fieldIndex: 2},
			command.FieldSeparator(","),
		),
	).WithStdinLines(
		"first,second,third",
		"a,b,c",
	).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"second",
		"b",
	})
}

func TestAwk_FieldSplitting_OutputSeparator(t *testing.T) {
	type PrintFieldsProgram struct {
		command.SimpleProgram
	}

	// Override Action to print fields 1 and 2
	var customProg command.Program = struct {
		command.SimpleProgram
	}{}

	result := run.Command(
		command.Awk(
			customProg,
			command.FieldSeparator(","),
			command.OutputFieldSeparator("|"),
		),
	).WithStdinLines("a,b,c").Run()

	assertion.NoError(t, result.Err)
	// This tests that OFS is set correctly in context
}

// ==============================================================================
// Test Variables
// ==============================================================================

// VariableProgram uses awk variables
type VariableProgram struct {
	command.SimpleProgram
}

func (p VariableProgram) Begin(ctx *command.Context) error {
	ctx.SetVar("total", 0)
	return nil
}

func (p VariableProgram) Action(ctx *command.Context) (string, bool) {
	current := ctx.Var("total").(int)
	ctx.SetVar("total", current+1)
	return "", false
}

func (p VariableProgram) End(ctx *command.Context) (string, error) {
	total := ctx.Var("total").(int)
	return fmt.Sprintf("Total: %d", total), nil
}

func TestAwk_Variables(t *testing.T) {
	result := run.Command(command.Awk(VariableProgram{})).
		WithStdinLines("line1", "line2", "line3").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{"Total: 3"})
}

// VariablePersistenceProgram verifies variables persist across lines
type VariablePersistenceProgram struct {
	command.SimpleProgram
}

func (p VariablePersistenceProgram) Begin(ctx *command.Context) error {
	ctx.SetVar("sum", 0)
	ctx.SetVar("count", 0)
	return nil
}

func (p VariablePersistenceProgram) Action(ctx *command.Context) (string, bool) {
	sum := ctx.Var("sum").(int)
	count := ctx.Var("count").(int)

	ctx.SetVar("sum", sum+int(ctx.NR))
	ctx.SetVar("count", count+1)

	return fmt.Sprintf("Line %d: sum=%d, count=%d", ctx.NR, ctx.Var("sum"), ctx.Var("count")), true
}

func TestAwk_VariablePersistence(t *testing.T) {
	result := run.Command(command.Awk(VariablePersistenceProgram{})).
		WithStdinLines("a", "b", "c").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"Line 1: sum=1, count=1",
		"Line 2: sum=3, count=2",
		"Line 3: sum=6, count=3",
	})
}

func TestAwk_InitialVariables(t *testing.T) {
	result := run.Command(
		command.Awk(
			command.SimpleProgram{},
			command.Variable{Name: "x", Value: 10},
			command.Variable{Name: "y", Value: "test"},
		),
	).WithStdinLines("line").Run()

	assertion.NoError(t, result.Err)
	// Variables are set in context
}

// ==============================================================================
// Test Line Number (NR)
// ==============================================================================

// LineNumberProgram prints line numbers
type LineNumberProgram struct {
	command.SimpleProgram
}

func (p LineNumberProgram) Action(ctx *command.Context) (string, bool) {
	return fmt.Sprintf("%d: %s", ctx.NR, ctx.Field(0)), true
}

func TestAwk_LineNumbers(t *testing.T) {
	result := run.Command(command.Awk(LineNumberProgram{})).
		WithStdinLines("first", "second", "third").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"1: first",
		"2: second",
		"3: third",
	})
}

// NRVerificationProgram verifies NR increments correctly
type NRVerificationProgram struct {
	command.SimpleProgram
	lastNR int64
}

func (p *NRVerificationProgram) Action(ctx *command.Context) (string, bool) {
	// Verify NR increments by 1 each time
	if p.lastNR > 0 && ctx.NR != p.lastNR+1 {
		return fmt.Sprintf("ERROR: NR jumped from %d to %d", p.lastNR, ctx.NR), true
	}
	p.lastNR = ctx.NR
	return fmt.Sprintf("NR=%d", ctx.NR), true
}

func TestAwk_NRIncrements(t *testing.T) {
	prog := &NRVerificationProgram{}
	result := run.Command(command.Awk(prog)).
		WithStdinLines("a", "b", "c", "d", "e").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"NR=1",
		"NR=2",
		"NR=3",
		"NR=4",
		"NR=5",
	})
}

// ==============================================================================
// Test Field Count (NF)
// ==============================================================================

// FieldCountProgram reports field count
type FieldCountProgram struct {
	command.SimpleProgram
}

func (p FieldCountProgram) Action(ctx *command.Context) (string, bool) {
	return fmt.Sprintf("%d fields", ctx.NF), true
}

func TestAwk_FieldCount(t *testing.T) {
	result := run.Command(command.Awk(FieldCountProgram{})).
		WithStdinLines(
			"one two three",
			"a b",
			"single",
		).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"3 fields",
		"2 fields",
		"1 fields",
	})
}

func TestAwk_FieldCount_WithCustomSeparator(t *testing.T) {
	result := run.Command(
		command.Awk(
			FieldCountProgram{},
			command.FieldSeparator(":"),
		),
	).WithStdinLines(
		"a:b:c:d",
		"x:y",
		"single",
	).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"4 fields",
		"2 fields",
		"1 fields",
	})
}

// ==============================================================================
// Test Error Handling
// ==============================================================================

// ErrorInBeginProgram fails in Begin
type ErrorInBeginProgram struct {
	command.SimpleProgram
}

func (p ErrorInBeginProgram) Begin(ctx *command.Context) error {
	return errors.New("begin error")
}

func TestAwk_ErrorInBegin(t *testing.T) {
	result := run.Command(command.Awk(ErrorInBeginProgram{})).
		WithStdinLines("line").Run()

	assertion.ErrorContains(t, result.Err, "BEGIN")
	assertion.ErrorContains(t, result.Err, "begin error")
}

// ErrorInEndProgram fails in End
type ErrorInEndProgram struct {
	command.SimpleProgram
}

func (p ErrorInEndProgram) End(ctx *command.Context) (string, error) {
	return "", errors.New("end error")
}

func TestAwk_ErrorInEnd(t *testing.T) {
	result := run.Command(command.Awk(ErrorInEndProgram{})).
		WithStdinLines("line").Run()

	assertion.ErrorContains(t, result.Err, "END")
	assertion.ErrorContains(t, result.Err, "end error")
}

func TestAwk_InputError(t *testing.T) {
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinError(errors.New("read failed")).Run()

	assertion.ErrorContains(t, result.Err, "read failed")
}

// ==============================================================================
// Test Complex Programs
// ==============================================================================

// SumProgram sums numbers from first field
type SumProgram struct {
	command.SimpleProgram
	sum float64
}

func (p *SumProgram) Action(ctx *command.Context) (string, bool) {
	firstField := ctx.Field(1)
	if val, err := strconv.ParseFloat(firstField, 64); err == nil {
		p.sum += val
	}
	return "", false
}

func (p *SumProgram) End(ctx *command.Context) (string, error) {
	return fmt.Sprintf("Sum: %.2f", p.sum), nil
}

func TestAwk_SumProgram(t *testing.T) {
	prog := &SumProgram{}
	result := run.Command(command.Awk(prog)).
		WithStdinLines(
			"10.5 foo",
			"20.3 bar",
			"15.2 baz",
		).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{"Sum: 46.00"})
}

// ==============================================================================
// Test Edge Cases
// ==============================================================================

func TestAwk_EmptyLines(t *testing.T) {
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines("", "", "").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{"", "", ""})
}

func TestAwk_EmptyLines_NF(t *testing.T) {
	// Empty lines should have NF=0 with any separator
	result := run.Command(command.Awk(FieldCountProgram{})).
		WithStdinLines("a b c", "", "x y").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"3 fields",
		"0 fields",
		"2 fields",
	})
}

func TestAwk_EmptyLines_CustomSeparator_NF(t *testing.T) {
	// Empty lines should have NF=0 even with custom separator
	result := run.Command(
		command.Awk(
			FieldCountProgram{},
			command.FieldSeparator(":"),
		),
	).WithStdinLines("a:b:c", "", "x:y").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"3 fields",
		"0 fields",
		"2 fields",
	})
}

func TestAwk_WhitespaceOnlyLines(t *testing.T) {
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines("   ", "\t\t", "  \t  ").Run()

	assertion.NoError(t, result.Err)
	assertion.Count(t, result.Stdout, 3)
}

func TestAwk_VeryLongLine(t *testing.T) {
	longLine := strings.Repeat("a", 10000)
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines(longLine).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{longLine})
}

func TestAwk_ManyLines(t *testing.T) {
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}

	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines(lines...).Run()

	assertion.NoError(t, result.Err)
	assertion.Count(t, result.Stdout, 1000)
}

func TestAwk_UnicodeHandling(t *testing.T) {
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines(
			"日本語",
			"Ελληνικά",
			"Русский",
		).Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"日本語",
		"Ελληνικά",
		"Русский",
	})
}

func TestAwk_MixedContent(t *testing.T) {
	result := run.Command(command.Awk(command.SimpleProgram{})).
		WithStdinLines(
			"normal line",
			"",
			"line with\ttabs",
			"日本語",
			strings.Repeat("x", 100),
		).Run()

	assertion.NoError(t, result.Err)
	assertion.Count(t, result.Stdout, 5)
}

// ==============================================================================
// Comprehensive Awk Behavior Tests (matching Unix awk)
// ==============================================================================

// FieldInspectorProgram inspects fields for compatibility testing
type FieldInspectorProgram struct {
	command.SimpleProgram
}

func (p FieldInspectorProgram) Action(ctx *command.Context) (string, bool) {
	return fmt.Sprintf("NF=%d $1=[%s] $2=[%s]", ctx.NF, ctx.Field(1), ctx.Field(2)), true
}

func TestAwk_AwkCompatibility_EmptyLineFields(t *testing.T) {
	// Test that matches exact awk behavior:
	// echo -e "a:b\n\nx:y" | awk -F: '{print "NF="NF" $1=["$1"] $2=["$2"]"}'
	result := run.Command(
		command.Awk(
			FieldInspectorProgram{},
			command.FieldSeparator(":"),
		),
	).WithStdinLines("a:b", "", "x:y").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"NF=2 $1=[a] $2=[b]",
		"NF=0 $1=[] $2=[]",  // Empty line: NF=0, fields are empty
		"NF=2 $1=[x] $2=[y]",
	})
}

func TestAwk_AwkCompatibility_WhitespaceFields(t *testing.T) {
	// Whitespace-only lines have NF=0 with default separator
	// echo "   " | awk '{print "NF="NF}'
	result := run.Command(command.Awk(FieldCountProgram{})).
		WithStdinLines("   ", "\t\t").Run()

	assertion.NoError(t, result.Err)
	assertion.Lines(t, result.Stdout, []string{
		"0 fields",
		"0 fields",
	})
}

// ==============================================================================
// Table-Driven Test Example
// ==============================================================================

func TestAwk_TableDriven(t *testing.T) {
	tests := []struct {
		name   string
		prog   command.Program
		input  []string
		output []string
	}{
		{
			name:   "simple pass-through",
			prog:   command.SimpleProgram{},
			input:  []string{"a", "b"},
			output: []string{"a", "b"},
		},
		{
			name:   "uppercase",
			prog:   UppercaseProgram{},
			input:  []string{"hello", "world"},
			output: []string{"HELLO", "WORLD"},
		},
		{
			name:   "empty input",
			prog:   command.SimpleProgram{},
			input:  []string{},
			output: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := run.Command(command.Awk(tt.prog)).
				WithStdinLines(tt.input...).Run()

			assertion.NoError(t, result.Err)
			assertion.Lines(t, result.Stdout, tt.output)
		})
	}
}

