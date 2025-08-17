package awk

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	localopt "github.com/yupsh/awk/opt"
	yup "github.com/yupsh/framework"
	"github.com/yupsh/framework/opt"
)

// Flags represents the configuration options for the awk command
type Flags = localopt.Flags

// Command implementation
type command opt.Inputs[string, Flags]

// Awk creates a new awk command with the given parameters
func Awk(parameters ...any) yup.Command {
	cmd := command(opt.Args[string, Flags](parameters...))
	// Set default field separator
	if cmd.Flags.FieldSeparator == "" {
		cmd.Flags.FieldSeparator = " "
	}
	return cmd
}

func (c command) Execute(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	// Get program from flags or first positional argument
	program := string(c.Flags.Program)
	if program == "" && len(c.Positional) > 0 {
		program = c.Positional[0]
	}

	if program == "" {
		fmt.Fprintln(stderr, "awk: missing program")
		return fmt.Errorf("missing program")
	}

	// Parse AWK program (very simplified)
	awkProgram, err := c.parseProgram(program)
	if err != nil {
		fmt.Fprintf(stderr, "awk: %v\n", err)
		return err
	}

	// Process files or stdin
	var files []string
	if string(c.Flags.Program) != "" {
		// Program was provided via flags, all positional args are files
		files = c.Positional
	} else if len(c.Positional) > 1 {
		// First positional arg is program, rest are files
		files = c.Positional[1:]
	} else {
		// Only program provided, no files - read from stdin
		files = []string{}
	}

	return yup.ProcessFilesWithContext(
		ctx, files, stdin, stdout, stderr,
		yup.FileProcessorOptions{
			CommandName:     "awk",
			ContinueOnError: true,
		},
		func(ctx context.Context, source yup.InputSource, output io.Writer) error {
			return c.processReader(ctx, source.Reader, output, awkProgram)
		},
	)
}

type AwkProgram struct {
	Pattern string
	Action  string
}

func (c command) parseProgram(program string) (*AwkProgram, error) {
	// Very simplified AWK parsing
	// Real AWK would have full lexer/parser

	if strings.Contains(program, "{") && strings.Contains(program, "}") {
		// Extract action
		start := strings.Index(program, "{")
		end := strings.LastIndex(program, "}")
		if start < end {
			pattern := strings.TrimSpace(program[:start])
			action := strings.TrimSpace(program[start+1 : end])
			return &AwkProgram{Pattern: pattern, Action: action}, nil
		}
	}

	// Treat as simple action
	return &AwkProgram{Pattern: "", Action: program}, nil
}

func (c command) processReader(ctx context.Context, reader io.Reader, output io.Writer, program *AwkProgram) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for yup.ScanWithContext(ctx, scanner) {
		lineNum++
		line := scanner.Text()

		// Split into fields
		var fields []string
		if string(c.Flags.FieldSeparator) == " " {
			fields = strings.Fields(line)
		} else {
			fields = strings.Split(line, string(c.Flags.FieldSeparator))
		}

		// Check pattern match (simplified)
		if c.matchesPattern(line, fields, program.Pattern) {
			result := c.executeAction(line, fields, lineNum, program.Action)
			if result != "" {
				fmt.Fprintln(output, result)
			}
		}
	}

	// Check if context was cancelled
	if err := yup.CheckContextCancellation(ctx); err != nil {
		return err
	}

	return scanner.Err()
}

func (c command) matchesPattern(line string, fields []string, pattern string) bool {
	if pattern == "" {
		return true // Empty pattern matches all lines
	}

	// Very simplified pattern matching
	// Real AWK would support regex, conditions, etc.
	return strings.Contains(line, pattern)
}

func (c command) executeAction(line string, fields []string, lineNum int, action string) string {
	// Very simplified action execution
	// Real AWK would have full expression evaluator

	switch action {
	case "print":
		return line
	case "print NF":
		return strconv.Itoa(len(fields))
	case "print NR":
		return strconv.Itoa(lineNum)
	case "print $0":
		return line
	case "print $1":
		if len(fields) > 0 {
			return fields[0]
		}
		return ""
	case "print $2":
		if len(fields) > 1 {
			return fields[1]
		}
		return ""
	default:
		// Try to handle print $N patterns
		if strings.HasPrefix(action, "print $") {
			fieldStr := action[7:]
			if fieldNum, err := strconv.Atoi(fieldStr); err == nil && fieldNum > 0 && fieldNum <= len(fields) {
				return fields[fieldNum-1]
			}
		}
		return line
	}
}
