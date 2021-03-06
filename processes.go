package empire

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"

	shellwords "github.com/mattn/go-shellwords"
	"github.com/remind101/empire/pkg/constraints"
)

// DefaultQuantities maps a process type to the default number of instances to
// run.
var DefaultQuantities = map[string]int{
	"web": 1,
}

// Command represents the actual shell command that gets executed for a given
// ProcessType.
type Command []string

// ParseCommand parses a string into a Command, taking quotes and other shell
// words into account.
func ParseCommand(command string) (Command, error) {
	return shellwords.Parse(command)
}

// MustParseCommand parses the string into a Command, panicing if there's an
// error. This method should only be used in tests for convenience.
func MustParseCommand(command string) Command {
	c, err := ParseCommand(command)
	if err != nil {
		panic(err)
	}
	return c
}

// Scan implements the sql.Scanner interface.
func (c *Command) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	var cmd Command
	if err := json.Unmarshal(bytes, &cmd); err != nil {
		return err
	}
	*c = cmd

	return nil
}

// Value implements the driver.Value interface.
func (c Command) Value() (driver.Value, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return driver.Value(raw), nil
}

// String returns the string reprsentation of the command.
func (c Command) String() string {
	return strings.Join([]string(c), " ")
}

// Process holds configuration information about a Process.
type Process struct {
	Command  Command              `json:"Command,omitempty"`
	Quantity int                  `json:"Quantity,omitempty"`
	Memory   constraints.Memory   `json:"Memory,omitempty"`
	CPUShare constraints.CPUShare `json:"CPUShare,omitempty"`
	Nproc    constraints.Nproc    `json:"Nproc,omitempty"`
}

// Constraints returns a constraints.Constraints from this Process definition.
func (p *Process) Constraints() Constraints {
	return Constraints{
		Memory:   p.Memory,
		CPUShare: p.CPUShare,
		Nproc:    p.Nproc,
	}
}

// SetConstraints sets the memory/cpu/nproc for this Process to the given
// constraints.
func (p *Process) SetConstraints(c Constraints) {
	p.Memory = c.Memory
	p.CPUShare = c.CPUShare
	p.Nproc = c.Nproc
}

// Formation represents a collection of named processes and their configuration.
type Formation map[string]Process

// Scan implements the sql.Scanner interface.
func (f *Formation) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	formation := make(Formation)
	if err := json.Unmarshal(bytes, &formation); err != nil {
		return err
	}
	*f = formation

	return nil
}

// Value implements the driver.Value interface.
func (f Formation) Value() (driver.Value, error) {
	if f == nil {
		return nil, nil
	}

	raw, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	return driver.Value(raw), nil
}

// Merge merges in the existing quantity and constraints from the old Formation
// into this Formation.
func (f Formation) Merge(other Formation) Formation {
	new := make(Formation)

	for name, p := range f {
		if existing, found := other[name]; found {
			// If the existing Formation already had a process
			// configuration for this process type, copy over the
			// instance count.
			p.Quantity = existing.Quantity
			p.SetConstraints(existing.Constraints())
		} else {
			p.Quantity = DefaultQuantities[name]
			p.SetConstraints(DefaultConstraints)
		}

		new[name] = p
	}

	return new
}
