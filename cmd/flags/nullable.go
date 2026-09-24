package flags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// The nullable values below are generic over the flag's underlying kind, so
// named API types (e.g. type EnvdVersion = string, type SandboxState string)
// can be bound directly without an explicit conversion at the call site.

type nullableBoolValue[T ~bool] struct {
	target **T
}

func (v *nullableBoolValue[T]) String() string {
	if *v.target == nil {
		return ""
	}
	return strconv.FormatBool(bool(**v.target))
}

func (v *nullableBoolValue[T]) Set(s string) error {
	parsed, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}

	value := T(parsed)
	*v.target = &value
	return nil
}

func (v *nullableBoolValue[T]) Type() string {
	return "bool"
}

func NullableBoolVarP[T ~bool](
	fs *pflag.FlagSet,
	target **T,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableBoolValue[T]{target: target}, name, shorthand, usage)

	fs.Lookup(name).NoOptDefVal = "true"
}

type nullableIntValue[T ~int] struct {
	target **T
}

func (v *nullableIntValue[T]) String() string {
	if *v.target == nil {
		return ""
	}
	return strconv.Itoa(int(**v.target))
}

func (v *nullableIntValue[T]) Set(s string) error {
	parsed, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("invalid int value %q: %w", s, err)
	}

	value := T(parsed)
	*v.target = &value
	return nil
}

func (v *nullableIntValue[T]) Type() string {
	return "int"
}

func NullableIntVarP[T ~int](
	fs *pflag.FlagSet,
	target **T,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableIntValue[T]{target: target}, name, shorthand, usage)
}

type nullableInt32Value[T ~int32] struct {
	target **T
}

func (v *nullableInt32Value[T]) String() string {
	if *v.target == nil {
		return ""
	}
	return strconv.FormatInt(int64(**v.target), 10)
}

func (v *nullableInt32Value[T]) Set(s string) error {
	parsed, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid int32 value %q: %w", s, err)
	}

	value := T(parsed)
	*v.target = &value
	return nil
}

func (v *nullableInt32Value[T]) Type() string {
	return "int32"
}

func NullableInt32VarP[T ~int32](
	fs *pflag.FlagSet,
	target **T,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableInt32Value[T]{target: target}, name, shorthand, usage)
}

type nullableInt64Value[T ~int64] struct {
	target **T
}

func (v *nullableInt64Value[T]) String() string {
	if *v.target == nil {
		return ""
	}
	return strconv.FormatInt(int64(**v.target), 10)
}

func (v *nullableInt64Value[T]) Set(s string) error {
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid int64 value %q: %w", s, err)
	}

	value := T(parsed)
	*v.target = &value
	return nil
}

func (v *nullableInt64Value[T]) Type() string {
	return "int64"
}

func NullableInt64VarP[T ~int64](
	fs *pflag.FlagSet,
	target **T,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableInt64Value[T]{target: target}, name, shorthand, usage)
}

type nullableUint32Value[T ~uint32] struct {
	target **T
}

func (v *nullableUint32Value[T]) String() string {
	if *v.target == nil {
		return ""
	}
	return strconv.FormatUint(uint64(**v.target), 10)
}

func (v *nullableUint32Value[T]) Set(s string) error {
	parsed, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid uint32 value %q: %w", s, err)
	}

	value := T(parsed)
	*v.target = &value
	return nil
}

func (v *nullableUint32Value[T]) Type() string {
	return "uint32"
}

func NullableUint32VarP[T ~uint32](
	fs *pflag.FlagSet,
	target **T,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableUint32Value[T]{target: target}, name, shorthand, usage)
}

type nullableStringValue[T ~string] struct {
	target **T
}

func (v *nullableStringValue[T]) String() string {
	if *v.target == nil {
		return ""
	}
	return string(**v.target)
}

func (v *nullableStringValue[T]) Set(s string) error {
	value := T(s)
	*v.target = &value
	return nil
}

func (v *nullableStringValue[T]) Type() string {
	return "string"
}

func NullableStringVarP[T ~string](
	fs *pflag.FlagSet,
	target **T,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableStringValue[T]{target: target}, name, shorthand, usage)
}

// nullableStringSliceValue binds a repeatable flag to a nullable slice field.
// Every occurrence of the flag appends a single value to the slice. The element
// type is generic as well, so slices of named string types (e.g.
// []api.SandboxState) can be bound too.
type nullableStringSliceValue[S ~[]E, E ~string] struct {
	target **S
}

func (v *nullableStringSliceValue[S, E]) String() string {
	if *v.target == nil {
		return ""
	}

	parts := make([]string, 0, len(**v.target))
	for _, value := range **v.target {
		parts = append(parts, string(value))
	}

	return strings.Join(parts, ",")
}

func (v *nullableStringSliceValue[S, E]) Set(s string) error {
	if *v.target == nil {
		slice := make(S, 0, 1)
		*v.target = &slice
	}

	**v.target = append(**v.target, E(s))

	return nil
}

func (v *nullableStringSliceValue[S, E]) Type() string {
	return "stringSlice"
}

// NullableStringSliceVarP registers a repeatable flag. The slice is allocated
// on first use, so the target stays nil when the flag is not provided.
func NullableStringSliceVarP[S ~[]E, E ~string](
	fs *pflag.FlagSet,
	target **S,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableStringSliceValue[S, E]{target: target}, name, shorthand, usage)
}

// nullableUint32SliceValue binds a repeatable flag to a nullable slice field.
// Every occurrence of the flag appends a single value to the slice.
type nullableUint32SliceValue[S ~[]uint32] struct {
	target **S
}

func (v *nullableUint32SliceValue[S]) String() string {
	if *v.target == nil {
		return ""
	}

	parts := make([]string, 0, len(**v.target))
	for _, value := range **v.target {
		parts = append(parts, strconv.FormatUint(uint64(value), 10))
	}

	return strings.Join(parts, ",")
}

func (v *nullableUint32SliceValue[S]) Set(s string) error {
	parsed, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid uint32 value %q: %w", s, err)
	}

	if *v.target == nil {
		slice := make(S, 0, 1)
		*v.target = &slice
	}

	**v.target = append(**v.target, uint32(parsed))

	return nil
}

func (v *nullableUint32SliceValue[S]) Type() string {
	return "uint32Slice"
}

// NullableUint32SliceVarP registers a repeatable flag. The slice is allocated
// on first use, so the target stays nil when the flag is not provided.
func NullableUint32SliceVarP[S ~[]uint32](
	fs *pflag.FlagSet,
	target **S,
	name, shorthand string,
	usage string,
) {
	fs.VarP(&nullableUint32SliceValue[S]{target: target}, name, shorthand, usage)
}
