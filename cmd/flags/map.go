package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

// The map values below are generic over the map type, so named API types such
// as api.EnvVars can be bound without an explicit conversion at the call site.

// stringMapValue binds a repeatable "key: value" flag to a map field.
type stringMapValue[M ~map[string]string] struct {
	target *M
}

func (v *stringMapValue[M]) String() string {
	return formatStringMap(*v.target)
}

func (v *stringMapValue[M]) Set(s string) error {
	key, value, err := parseStringMapEntry(s)
	if err != nil {
		return err
	}

	if *v.target == nil {
		*v.target = make(M)
	}

	(*v.target)[key] = value

	return nil
}

func (v *stringMapValue[M]) Type() string {
	return "stringMap"
}

// StringMapVarP registers a repeatable "key: value" flag. The map is allocated
// on first use, so the target stays nil when the flag is not provided.
func StringMapVarP[M ~map[string]string](
	fs *pflag.FlagSet,
	target *M,
	name, shorthand, usage string,
) {
	fs.VarP(&stringMapValue[M]{target: target}, name, shorthand, usage)
}

// nullableStringMapValue binds a repeatable "key: value" flag to a nullable map
// field.
type nullableStringMapValue[M ~map[string]string] struct {
	target **M
}

func (v *nullableStringMapValue[M]) String() string {
	if *v.target == nil {
		return ""
	}

	return formatStringMap(**v.target)
}

func (v *nullableStringMapValue[M]) Set(s string) error {
	key, value, err := parseStringMapEntry(s)
	if err != nil {
		return err
	}

	if *v.target == nil {
		m := make(M)
		*v.target = &m
	}

	(**v.target)[key] = value

	return nil
}

func (v *nullableStringMapValue[M]) Type() string {
	return "stringMap"
}

// NullableStringMapVarP registers a repeatable "key: value" flag. The map is
// allocated on first use, so the target stays nil when the flag is not
// provided.
func NullableStringMapVarP[M ~map[string]string](
	fs *pflag.FlagSet,
	target **M,
	name, shorthand, usage string,
) {
	fs.VarP(&nullableStringMapValue[M]{target: target}, name, shorthand, usage)
}

// parseStringMapEntry splits one "key: value" occurrence of a map flag.
func parseStringMapEntry(s string) (string, string, error) {
	key, value, ok := strings.Cut(s, ":")
	if !ok {
		return "", "", fmt.Errorf("invalid map value %q: expected key: value", s)
	}

	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)

	if key == "" {
		return "", "", fmt.Errorf("invalid map value %q: key cannot be empty", s)
	}

	return key, value, nil
}

// formatStringMap renders a map flag's current value.
func formatStringMap[M ~map[string]string](m M) string {
	parts := make([]string, 0, len(m))
	for key, value := range m {
		parts = append(parts, key+": "+value)
	}

	return strings.Join(parts, ", ")
}
