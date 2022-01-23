package controllers

import "errors"

type Error string

const (
	ObjectUpdated Error = "object-updated"
)

func (e Error) Error() string {
	return string(e)
}

func (e Error) Is(target error) bool {
	var targetError Error
	if errors.As(target, &targetError) {
		if string(e) == string(targetError) {
			return true
		}
	}
	return false
}
