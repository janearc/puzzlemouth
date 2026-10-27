package numbers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// wire is a numbers file as YAML spells it, for go.uber.org/config to
// fill by name.
type wire struct {
	//ninja:vow gate the yaml reader fills this by its name
	Numbers []entry `yaml:"numbers"`
}

// entry is one number as YAML spells it.
type entry struct {
	//ninja:vow gate the yaml reader fills this by its name
	Name string `yaml:"name"`
	//ninja:vow gate the yaml reader fills this by its name
	Value *float64 `yaml:"value"`
	//ninja:vow gate the yaml reader fills this by its name
	Unit string `yaml:"unit"`
	//ninja:vow gate the yaml reader fills this by its name
	Says string `yaml:"says"`
	//ninja:vow gate the yaml reader fills this by its name
	Kind Kind `yaml:"kind"`
	//ninja:vow gate the yaml reader fills this by its name
	Source *sourced `yaml:"source"`
	//ninja:vow gate the yaml reader fills this by its name
	By string `yaml:"by"`
}

// sourced is a measured number's source as YAML spells it.
type sourced struct {
	//ninja:vow gate the yaml reader fills this by its name
	Key string `yaml:"key"`
	//ninja:vow gate the yaml reader fills this by its name
	Where string `yaml:"where"`
	//ninja:vow gate the yaml reader fills this by its name
	Page int `yaml:"page"`
	//ninja:vow gate the yaml reader fills this by its name
	Text string `yaml:"text"`
}

const (
	// fromPrefix asks ParseInt to take the base from the prefix, as YAML
	// does for 0x10 and 010. It's defined by strconv.ParseInt: base 0.
	fromPrefix = 0
	// bits is the size a number is read at. It's defined: a float64 is
	// 64 bits.
	bits = 64
)

// readAsAnother is whether the YAML reader may have changed the name. It
// reads yes as true and 010 as 8. So a name that is true, false or a
// number may not be what was written.
func readAsAnother(name string) bool {
	_, intErr := strconv.ParseInt(name, fromPrefix, bits)
	_, floatErr := strconv.ParseFloat(name, bits)
	counted := strings.ContainsAny(name, "0123456789") &&
		(intErr == nil || floatErr == nil)
	return name == "true" || name == "false" || counted
}

// number is the entry as a Number, or what it lacks.
func (written entry) number() (Number, error) {
	if written.Name == "" {
		return Number{}, errors.New("a number has no name")
	}
	if readAsAnother(written.Name) {
		return Number{}, fmt.Errorf("%s: this name is what the YAML "+
			"reader makes of another, as true of yes or 8 of 010; "+
			"call the number something else", written.Name)
	}
	if written.Value == nil || written.Says == "" {
		return Number{}, fmt.Errorf("%s: a number needs a value and "+
			"says what it is", written.Name)
	}
	made := Number{name: written.Name, value: *written.Value,
		unit: written.Unit, says: written.Says, kind: written.Kind}
	switch written.Kind {
	case Measured:
		return written.measured(made)
	case Picked:
		return written.picked(made)
	}
	return Number{}, fmt.Errorf("%s: kind is measured or picked, not %q",
		written.Name, written.Kind)
}

// measured is a measured entry's number, which needs its whole source
// and no picker.
func (written entry) measured(made Number) (Number, error) {
	from := written.Source
	if from == nil || from.Key == "" || from.Where == "" ||
		from.Text == "" || written.By != "" {
		return Number{}, fmt.Errorf("%s: a measured number names its "+
			"source's key, where in it, and the text there, and "+
			"no one picked it", written.Name)
	}
	made.from = Source{key: from.Key, where: from.Where, text: from.Text,
		page: from.Page}
	return made, nil
}

// picked is a picked entry's number, which needs who picked it and no
// source.
func (written entry) picked(made Number) (Number, error) {
	if written.By == "" || written.Source != nil {
		return Number{}, fmt.Errorf("%s: a picked number says who "+
			"picked it and when, and names no source", written.Name)
	}
	made.by = written.By
	return made, nil
}
