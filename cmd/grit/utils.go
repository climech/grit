package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/climech/grit/app"
)

func mustParseSelector(s string) app.Selector {
	sel, err := app.ParseSelector(s)
	if err != nil {
		die(capitalize(err.Error()))
	}
	return sel
}

func die(a ...interface{}) {
	msg := fmt.Sprint(a...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(os.Stderr, msg)
	os.Exit(1)
}

func errf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(os.Stderr, msg)
}

func dief(format string, a ...interface{}) {
	errf(format, a...)
	os.Exit(1)
}

func capitalize(s string) string {
	if len(s) > 0 {
		return strings.ToUpper(string(s[0])) + s[1:]
	}
	return s
}
