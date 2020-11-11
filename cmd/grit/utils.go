package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

func die(a ...interface{}) {
	msg := fmt.Sprint(a...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(os.Stderr, msg)
	os.Exit(1)
}

func dief(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(os.Stderr, msg)
	os.Exit(1)
}

func fmtCreate(s string) string {
	cyan := color.New(color.FgCyan).SprintFunc()
	return fmt.Sprintf("[%s] %s", cyan("++"), s)
}

func fmtDelete(s string) string {
	red := color.New(color.FgRed).SprintFunc()
	return fmt.Sprintf("[%s] %s", red("--"), s)
}

func fmtUpdate(s string) string {
	cyan := color.New(color.FgCyan).SprintFunc()
	return fmt.Sprintf("[%s] %s", cyan("**"), s)
}

func capitalize(s string) string {
	if len(s) > 0 {
		return strings.ToUpper(string(s[0])) + s[1:]
	}
	return s
}