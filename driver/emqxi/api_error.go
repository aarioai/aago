package emqxi

import "github.com/aarioai/aago/aa/ae"

type EmqxApiError struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

func NewError(err EmqxApiError) *ae.Error {
	if err.Code == "" {
		return nil
	}
	return ae.NewErrorf("code: %s, reason: %s", err.Code, err.Reason)
}
