package awk_test

import (
	"context"
	"os"
	"strings"

	"github.com/yupsh/awk"
	"github.com/yupsh/awk/opt"
)

func ExampleAwk() {
	ctx := context.Background()
	input := strings.NewReader("one two three\nfour five six\n")

	cmd := awk.Awk("{print $1}")
	cmd.Execute(ctx, input, os.Stdout, os.Stderr)
	// Output: one
	// four
}

func ExampleAwk_fieldSeparator() {
	ctx := context.Background()
	input := strings.NewReader("a,b,c\nd,e,f\n")

	cmd := awk.Awk("{print $2}", opt.FieldSeparator(","))
	cmd.Execute(ctx, input, os.Stdout, os.Stderr)
	// Output: b
	// e
}
