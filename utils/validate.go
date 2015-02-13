package utils

import (
	"net"
	"strconv"

	"github.com/techjanitor/pram-post/config"
)

// Validate will check string length
type Validate struct {
	Input string
	Max   int
	Min   int
}

// Parse parameters from requests to see if they are uint or too huge
func ValidateParam(param string) (id uint, err error) {
	pid, err := strconv.ParseUint(param, 10, 32)
	if err != nil {
		return
	} else if id > config.Settings.Limits.ParamMaxSize {
		return
	}
	id = uint(pid)

	return
}

// Parse ip and return true if it cant be parsed
func ValidateIP(ip string) bool {
	pip := net.ParseIP(ip)
	if pip == nil {
		return true
	}

	return false

}

// MaxLength checks string for length
func (v *Validate) MaxLength() bool {
	return len(v.Input) > v.Max
}

// MinLength checks string for length
func (v *Validate) MinLength() bool {
	return len(v.Input) < v.Min && len(v.Input) != 0
}

// IsEmpty checks to see if string is empty
func (v *Validate) IsEmpty() bool {
	return v.Input == ""
}
