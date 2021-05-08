package reviewtrigger

import (
	"errors"
	"strings"
)

func newErrors() *multiError {
	return &multiError{
		es: []string{},
	}
}

type multiError struct {
	es []string
}

func (e *multiError) add(s string) {
	e.es = append(e.es, s)
}

func (e *multiError) addError(err error) {
	e.es = append(e.es, err.Error())
}

func (e *multiError) err() error {
	if len(e.es) == 0 {
		return nil
	}
	return errors.New(strings.Join(e.es, ". "))
}
