# yup.awk

A Go-native awk implementation for the yupsh framework. Unlike traditional awk which uses string-based scripts, this implementation uses Go functions and interfaces for type-safe, composable text processing.

## Overview

The awk command provides awk-style text processing with:
- **BEGIN/END blocks** for initialization and finalization
- **Pattern matching** with conditions
- **Field splitting** with customizable separators
- **Variables** for stateful processing
- **Line numbers** and field counts (NR, NF)
- Full access to Go's type system and standard library

## Basic Usage

### Simple Field Printing

Print a specific field from each line:

```go
package main

import (
    "github.com/yupsh/awk"
    "github.com/yupsh/echo"
    "github.com/yupsh/framework"
    "github.com/yupsh/pipe"
)

type printFieldProgram struct {
    awk.SimpleProgram
    fieldNum int
}

func (p printFieldProgram) Action(ctx *awk.Context) (string, bool) {
    return ctx.Field(p.fieldNum), true
}

func main() {
    // echo "one two three" | awk '{print $2}'
    program := printFieldProgram{fieldNum: 2}
    yup.Run(pipe.Pipeline(
        echo.Echo("one two three"),
        awk.Awk(program),
    ))
}
```

## Program Interface

The `Program` interface defines four methods that correspond to awk's execution model:

```go
type Program interface {
    Begin(ctx *Context) error                    // Called once before processing
    Condition(ctx *Context) bool                 // Called for each line
    Action(ctx *Context) (output string, emit bool)  // Called when condition is true
    End(ctx *Context) (output string, err error)     // Called once after processing
}
```

### SimpleProgram

Use `SimpleProgram` as a base to avoid implementing all methods:

```go
type myProgram struct {
    awk.SimpleProgram  // Provides default implementations
}

// Only override what you need
func (p myProgram) Action(ctx *awk.Context) (string, bool) {
    return ctx.Field(0), true  // Print whole line
}
```

## Context API

The `Context` provides access to awk's execution environment:

### Fields

```go
// Field returns a field by index (0 = whole line, 1 = first field, etc.)
field := ctx.Field(1)

// SetField modifies a field
ctx.SetField(1, "newvalue")

// Access fields array directly
allFields := ctx.Fields  // []string
```

### Built-in Variables

```go
ctx.NR   // Current line number (1-based)
ctx.NF   // Number of fields in current line
ctx.FS   // Input field separator
ctx.OFS  // Output field separator
ctx.RS   // Record separator
```

### User Variables

```go
// Set a variable
ctx.SetVar("sum", 100)

// Get a variable
sum := ctx.Var("sum").(int)
```

### Helper Methods

```go
// Print formats values with OFS separator
output := ctx.Print(field1, field2, field3)
```

## Examples

### BEGIN and END Blocks

Use BEGIN for initialization and END for final output:

```go
type sumProgram struct {
    awk.SimpleProgram
}

func (p sumProgram) Begin(ctx *awk.Context) error {
    ctx.SetVar("sum", 0)
    return nil
}

func (p sumProgram) Action(ctx *awk.Context) (string, bool) {
    if val, err := strconv.Atoi(ctx.Field(1)); err == nil {
        sum := ctx.Var("sum").(int)
        ctx.SetVar("sum", sum+val)
    }
    return "", false  // Don't emit output per line
}

func (p sumProgram) End(ctx *awk.Context) (string, error) {
    return fmt.Sprintf("Sum: %d", ctx.Var("sum")), nil
}

// Usage: echo -e "10\n20\n30" | awk 'BEGIN {sum=0} {sum+=$1} END {print sum}'
program := sumProgram{}
yup.Run(pipe.Pipeline(
    echo.Echo("10\n20\n30"),
    awk.Awk(program),
))
// Output: Sum: 60
```

### Conditional Processing

Use `Condition` to filter which lines are processed:

```go
type grepProgram struct {
    awk.SimpleProgram
    pattern string
}

func (p grepProgram) Condition(ctx *awk.Context) bool {
    return strings.Contains(ctx.Field(0), p.pattern)
}

func (p grepProgram) Action(ctx *awk.Context) (string, bool) {
    return ctx.Field(0), true
}

// Usage: echo -e "apple\nbanana\napricot" | awk '/^ap/'
program := grepProgram{pattern: "ap"}
yup.Run(pipe.Pipeline(
    echo.Echo("apple\nbanana\napricot"),
    awk.Awk(program),
))
// Output:
// apple
// apricot
```

### Field Separators

Specify custom input and output field separators:

```go
program := printFieldProgram{fieldNum: 2}

// Input separator
yup.Run(pipe.Pipeline(
    echo.Echo("one:two:three"),
    awk.Awk(program, awk.FieldSeparator(":")),
))
// Output: two

// Output separator
yup.Run(pipe.Pipeline(
    echo.Echo("one two three"),
    awk.Awk(program,
        awk.FieldSeparator(" "),
        awk.OutputFieldSeparator(",")),
))
```

### Line Numbers

Access line numbers via `ctx.NR`:

```go
type lineNumberProgram struct {
    awk.SimpleProgram
}

func (p lineNumberProgram) Action(ctx *awk.Context) (string, bool) {
    return fmt.Sprintf("%d: %s", ctx.NR, ctx.Field(0)), true
}

// Usage: awk '{print NR": "$0}'
program := lineNumberProgram{}
yup.Run(pipe.Pipeline(
    echo.Echo("first\nsecond\nthird"),
    awk.Awk(program),
))
// Output:
// 1: first
// 2: second
// 3: third
```

### Complex Text Processing

Combine all features for sophisticated processing:

```go
type statsProgram struct {
    awk.SimpleProgram
}

func (p statsProgram) Begin(ctx *awk.Context) error {
    ctx.SetVar("count", 0)
    ctx.SetVar("sum", 0.0)
    ctx.SetVar("max", 0.0)
    return nil
}

func (p statsProgram) Action(ctx *awk.Context) (string, bool) {
    if val, err := strconv.ParseFloat(ctx.Field(1), 64); err == nil {
        count := ctx.Var("count").(int)
        sum := ctx.Var("sum").(float64)
        max := ctx.Var("max").(float64)

        ctx.SetVar("count", count+1)
        ctx.SetVar("sum", sum+val)
        if val > max {
            ctx.SetVar("max", val)
        }
    }
    return "", false
}

func (p statsProgram) End(ctx *awk.Context) (string, error) {
    count := ctx.Var("count").(int)
    sum := ctx.Var("sum").(float64)
    max := ctx.Var("max").(float64)

    avg := sum / float64(count)
    return fmt.Sprintf("Count: %d, Sum: %.2f, Max: %.2f, Avg: %.2f",
        count, sum, max, avg), nil
}
```

## Flags

Available flags for the `Awk` function:

### FieldSeparator

Set the input field separator (default: space/whitespace):

```go
awk.Awk(program, awk.FieldSeparator(":"))
```

### OutputFieldSeparator

Set the output field separator (default: space):

```go
awk.Awk(program, awk.OutputFieldSeparator(","))
```

### Variable

Initialize variables before BEGIN (supports any type):

```go
awk.Awk(program,
    awk.Variable{Name: "threshold", Value: 100},      // int
    awk.Variable{Name: "prefix", Value: "LOG:"},      // string
    awk.Variable{Name: "ratio", Value: 0.75},         // float64
    awk.Variable{Name: "enabled", Value: true},       // bool
)
```

## Design Philosophy

This awk implementation differs from traditional awk in several key ways:

1. **Type Safety**: Uses Go's type system instead of string parsing
2. **Composability**: Programs are Go structs that can embed logic and state
3. **Testability**: Programs can be unit tested independently
4. **IDE Support**: Full autocompletion and refactoring support
5. **Performance**: No string parsing overhead at runtime

Traditional awk:
```bash
awk 'BEGIN {sum=0} {sum+=$1} END {print sum}'
```

yup.awk:
```go
type sumProgram struct {
    awk.SimpleProgram
}
func (p sumProgram) Begin(ctx *awk.Context) error { ... }
func (p sumProgram) Action(ctx *awk.Context) (string, bool) { ... }
func (p sumProgram) End(ctx *awk.Context) (string, error) { ... }
```

## Pattern Matching

Since you have access to the full Go standard library, pattern matching is more powerful:

```go
import "regexp"

type regexProgram struct {
    awk.SimpleProgram
    re *regexp.Regexp
}

func (p regexProgram) Condition(ctx *awk.Context) bool {
    return p.re.MatchString(ctx.Field(0))
}
```

## Advanced Features

### Stateful Processing

Programs can maintain arbitrary state:

```go
type statefulProgram struct {
    awk.SimpleProgram
    previous string
    buffer   []string
}

func (p *statefulProgram) Action(ctx *awk.Context) (string, bool) {
    current := ctx.Field(0)
    if current != p.previous {
        p.buffer = append(p.buffer, current)
        p.previous = current
    }
    return "", false
}
```

### Error Handling

Return errors from any method:

```go
func (p myProgram) Begin(ctx *awk.Context) error {
    if _, err := os.Open("config.txt"); err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    return nil
}
```

### Multiple Output Lines

Emit multiple lines from a single input:

```go
func (p myProgram) Action(ctx *awk.Context) (string, bool) {
    // Process field and emit multiple times
    for i := 0; i < 3; i++ {
        // Note: This will emit once per call
        // For true multi-line output, accumulate in Action and emit in End
    }
    return ctx.Field(0), true
}
```

## See Also

- [yupsh framework](../framework/README.md)
- [Example tests](../examples/commands/awk/command_test.go)

