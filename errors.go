// Package errors provides simple error handling primitives.
//
// The traditional error handling idiom in Go is roughly akin to
//
//     if err != nil {
//             return err
//     }
//
// which when applied recursively up the call stack results in error reports
// without context or debugging information. The errors package allows
// programmers to add context to the failure path in their code in a way
// that does not destroy the original value of the error.
//
// Adding context to an error
//
// The errors.Wrap function returns a new error that adds context to the
// original error by recording a stack trace at the point Wrap is called,
// together with the supplied message. For example
//
//     _, err := ioutil.ReadAll(r)
//     if err != nil {
//             return errors.Wrap(err, "read failed")
//     }
//
// If additional control is required, the errors.WithStack and
// errors.WithMessage functions destructure errors.Wrap into its component
// operations: annotating an error with a stack trace and with a message,
// respectively.
//
// Retrieving the cause of an error
//
// Using errors.Wrap constructs a stack of errors, adding context to the
// preceding error. Depending on the nature of the error it may be necessary
// to reverse the operation of errors.Wrap to retrieve the original error
// for inspection. Any error value which implements this interface
//
//     type causer interface {
//             Cause() error
//     }
//
// can be inspected by errors.Cause. errors.Cause will recursively retrieve
// the topmost error that does not implement causer, which is assumed to be
// the original cause. For example:
//
//     switch err := errors.Cause(err).(type) {
//     case *MyError:
//             // handle specifically
//     default:
//             // unknown error
//     }
//
// Although the causer interface is not exported by this package, it is
// considered a part of its stable public interface.
//
// Formatted printing of errors
//
// All error values returned from this package implement fmt.Formatter and can
// be formatted by the fmt package. The following verbs are supported:
//
//     %s    print the error. If the error has a Cause it will be
//           printed recursively.
//     %v    see %s
//     %+v   extended format. Each Frame of the error's StackTrace will
//           be printed in detail.
//
// Retrieving the stack trace of an error or wrapper
//
// New, Errorf, Wrap, and Wrapf record a stack trace at the point they are
// invoked. This information can be retrieved with the following interface:
//
//     type stackTracer interface {
//             StackTrace() errors.StackTrace
//     }
//
// The returned errors.StackTrace type is defined as
//
//     type StackTrace []Frame
//
// The Frame type represents a call site in the stack trace. Frame supports
// the fmt.Formatter interface that can be used for printing information about
// the stack trace of this error. For example:
//
//     if err, ok := err.(stackTracer); ok {
//             for _, f := range err.StackTrace() {
//                     fmt.Printf("%+s:%d\n", f, f)
//             }
//     }
//
// Although the stackTracer interface is not exported by this package, it is
// considered a part of its stable public interface.
//
// See the documentation for Frame.Format for more details.
package errors

import (
	"errors"
	"fmt"
	"io"
)

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
func New(message string) error {
	return formatted{withStack{
		error: errors.New(message),
		stack: callers(0),
	}}
}

// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
// Errorf also records the stack trace at the point it was called.
func Errorf(format string, args ...interface{}) error {
	return formatted{withStack{
		error: fmt.Errorf(format, args...),
		stack: callers(0),
	}}
}

// WithStack is an alias for EnsureStack. Deprecated.
func WithStack(err error) error {
	return ensureStack(err)
}

// EnsureStack ensures err is annotated with a stack trace. In case it is not,
// it is annotated with a stack trace at the point EnsureStack was called.
// In case it already had a stack trace, err is returned as is.
// If err is nil, EnsureStack returns nil.
func EnsureStack(err error) error {
	return ensureStack(err)
}

func ensureStack(err error) error {
	if err == nil {
		return nil
	}
	var st interface {
		error
		StackTrace() StackTrace
	}
	if As(err, &st) {
		return formatted{err}
	}
	return formatted{withStack{
		err,
		callers(1),
	}}
}

type withStack struct {
	error
	*stack
}

func (w withStack) Cause() error { return w.error }

// Unwrap provides compatibility for Go 1.13 error chains.
func (w withStack) Unwrap() error { return w.error }

// Wrap returns an error annotating err with a stack trace
// at the point Wrap is called, and the supplied message.
// If err is nil, Wrap returns nil.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return formatted{fmt.Errorf("%s: %w", message, ensureStack(err))}
}

// Wrapf returns an error annotating err with a stack trace
// at the point Wrapf is called, and the format specifier.
// If err is nil, Wrapf returns nil.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return formatted{fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), ensureStack(err))}
}

type formatted struct {
	error
}

func (f formatted) Cause() error { return f.error }

func (f formatted) Unwrap() error { return f.error }

func (f formatted) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, f.error.Error())
			var st interface {
				Format(fmt.State, rune)
				StackTrace() StackTrace
			}
			if As(f.error, &st) {
				st.Format(s, verb)
			}
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, f.error.Error())
	case 'q':
		fmt.Fprintf(s, "%q", f.error.Error())
	}
}

// WithMessage annotates err with a new message.
// If err is nil, WithMessage returns nil.
func WithMessage(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// WithMessagef annotates err with the format specifier.
// If err is nil, WithMessagef returns nil.
func WithMessagef(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// Cause calls Unwrap on err repeatedly, until the error has a StackTrace()
// or does not implement Unwrap.
func Cause(err error) error {
	for {
		var st interface {
			error
			StackTrace() StackTrace
		}
		if As(err, &st) {
			return st
		}
		e := Unwrap(err)
		if e == nil {
			return err
		}
		err = e
	}
}
