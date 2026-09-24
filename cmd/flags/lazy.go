package flags

import (
	"github.com/spf13/pflag"
)

// lazyValue decorates the pflag.Value of a regularly registered flag: the
// flag writes to a standalone staging variable, and every successful Set
// allocates the owning struct and copies the staged value into it. The owner
// therefore stays nil unless one of its flags is actually provided.
type lazyValue[T any] struct {
	inner pflag.Value
	owner **T
	apply func(*T)
}

func (v *lazyValue[T]) String() string {
	return v.inner.String()
}

func (v *lazyValue[T]) Set(s string) error {
	if err := v.inner.Set(s); err != nil {
		return err
	}

	if *v.owner == nil {
		*v.owner = new(T)
	}

	v.apply(*v.owner)

	return nil
}

func (v *lazyValue[T]) Type() string {
	return v.inner.Type()
}

// LazyVarP registers a flag through register, which may be any regular
// registration call (pflag's own BoolVarP, this package's NullableStringSliceVarP,
// ...) binding a staging variable, and then makes it feed owner: apply copies
// the staged value into the struct, which is allocated on first use. The
// registered flag is returned so callers can tweak it further.
func LazyVarP[T any](
	fs *pflag.FlagSet,
	owner **T,
	register func(fs *pflag.FlagSet, name, shorthand, usage string),
	apply func(*T),
	name, shorthand, usage string,
) *pflag.Flag {
	register(fs, name, shorthand, usage)

	flag := fs.Lookup(name)
	flag.Value = &lazyValue[T]{inner: flag.Value, owner: owner, apply: apply}

	return flag
}

// The helpers below wrap LazyVarP for the common field kinds: they own the
// staging variable, so a call site only has to say which field of the owning
// struct the flag feeds. assign runs on every successful Set, which is also
// where a nested struct can be allocated in turn.

// LazyBoolVarP registers a bool flag feeding a non-pointer field.
func LazyBoolVarP[T any, B ~bool](
	fs *pflag.FlagSet,
	owner **T,
	assign func(*T, B),
	name, shorthand, usage string,
) {
	var staged bool

	LazyVarP(
		fs,
		owner,
		func(fs *pflag.FlagSet, name, shorthand, usage string) {
			fs.BoolVarP(&staged, name, shorthand, false, usage)
		},
		func(target *T) { assign(target, B(staged)) },
		name, shorthand, usage,
	)
}

// LazyStringVarP registers a string flag feeding a non-pointer field.
func LazyStringVarP[T any, S ~string](
	fs *pflag.FlagSet,
	owner **T,
	assign func(*T, S),
	name, shorthand, usage string,
) {
	var staged string

	LazyVarP(
		fs,
		owner,
		func(fs *pflag.FlagSet, name, shorthand, usage string) {
			fs.StringVarP(&staged, name, shorthand, "", usage)
		},
		func(target *T) { assign(target, S(staged)) },
		name, shorthand, usage,
	)
}

// LazyNullableBoolVarP registers a bool flag feeding a nullable field.
func LazyNullableBoolVarP[T any, B ~bool](
	fs *pflag.FlagSet,
	owner **T,
	assign func(*T, *B),
	name, shorthand, usage string,
) {
	var staged *B

	LazyVarP(
		fs,
		owner,
		func(fs *pflag.FlagSet, name, shorthand, usage string) {
			NullableBoolVarP(fs, &staged, name, shorthand, usage)
		},
		func(target *T) { assign(target, staged) },
		name, shorthand, usage,
	)
}

// LazyNullableStringVarP registers a string flag feeding a nullable field.
func LazyNullableStringVarP[T any, S ~string](
	fs *pflag.FlagSet,
	owner **T,
	assign func(*T, *S),
	name, shorthand, usage string,
) {
	var staged *S

	LazyVarP(
		fs,
		owner,
		func(fs *pflag.FlagSet, name, shorthand, usage string) {
			NullableStringVarP(fs, &staged, name, shorthand, usage)
		},
		func(target *T) { assign(target, staged) },
		name, shorthand, usage,
	)
}

// LazyNullableStringSliceVarP registers a repeatable flag feeding a nullable
// slice field.
func LazyNullableStringSliceVarP[T any, S ~[]E, E ~string](
	fs *pflag.FlagSet,
	owner **T,
	assign func(*T, *S),
	name, shorthand, usage string,
) {
	var staged *S

	LazyVarP(
		fs,
		owner,
		func(fs *pflag.FlagSet, name, shorthand, usage string) {
			NullableStringSliceVarP(fs, &staged, name, shorthand, usage)
		},
		func(target *T) { assign(target, staged) },
		name, shorthand, usage,
	)
}

// LazyNullableUint32SliceVarP registers a repeatable flag feeding a nullable
// slice field.
func LazyNullableUint32SliceVarP[T any, S ~[]uint32](
	fs *pflag.FlagSet,
	owner **T,
	assign func(*T, *S),
	name, shorthand, usage string,
) {
	var staged *S

	LazyVarP(
		fs,
		owner,
		func(fs *pflag.FlagSet, name, shorthand, usage string) {
			NullableUint32SliceVarP(fs, &staged, name, shorthand, usage)
		},
		func(target *T) { assign(target, staged) },
		name, shorthand, usage,
	)
}
