package errors

import (
	"fmt"
	"strings"
)

type ErrorAggregator struct {
	delim  string
	errors []error
}

func NewAggregator(delim string) *ErrorAggregator {
	return &ErrorAggregator{
		delim:  delim,
		errors: []error{},
	}
}

func (e *ErrorAggregator) Error() string {
	sb := strings.Builder{}
	for i, err := range e.errors {
		sb.WriteString(err.Error())
		if i != len(e.errors)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (e *ErrorAggregator) Add(err error) {
	e.errors = append(e.errors, err)
}

func (e *ErrorAggregator) Addf(format string, args ...any) {
	e.Add(fmt.Errorf(format, args...))
}

func (e *ErrorAggregator) HasError() bool {
	return len(e.errors) > 0
}
