package utils

import "fmt"

// CLIError represents an error that can be nicely formatted for CLI output
type CLIError struct {
	Message string
	Err     error
}

func (e *CLIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// NewError creates a new CLIError
func NewError(message string, err error) error {
	return &CLIError{
		Message: message,
		Err:     err,
	}
}

// HandleError prints the error and returns it
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	if cliErr, ok := err.(*CLIError); ok {
		PrintError("%s", cliErr.Error())
	} else {
		PrintError("%s", err.Error())
	}
	return err
}
