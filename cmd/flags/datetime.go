package flags

import (
	"time"

	"github.com/spf13/pflag"
	"github.com/ucloud/ucloud-sandbox-cli/internal/datetime"
)

// nullableDatetimeValue binds a flexible datetime flag to a nullable time
// field. The accepted formats are the ones internal/datetime parses, e.g.
// "15:04", "06-23 15:04" and "2025-07-23 15:04".
type nullableDatetimeValue struct {
	target **time.Time
}

func (v *nullableDatetimeValue) String() string {
	if *v.target == nil {
		return ""
	}

	return (*v.target).Format(time.DateTime)
}

func (v *nullableDatetimeValue) Set(s string) error {
	parsed, err := datetime.Parse(s)
	if err != nil {
		return err
	}

	*v.target = &parsed

	return nil
}

func (v *nullableDatetimeValue) Type() string {
	return "datetime"
}

// NullableDatetimeVarP registers a datetime flag. The target stays nil when the
// flag is not provided.
func NullableDatetimeVarP(
	fs *pflag.FlagSet,
	target **time.Time,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableDatetimeValue{target: target}, name, shorthand, usage)
}
