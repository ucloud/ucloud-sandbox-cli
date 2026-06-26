package table

import (
	"fmt"
	"reflect"
	"time"

	"github.com/dustin/go-humanize"
)

// formatter converts a single struct field value to its table cell string.
type formatter func(reflect.Value) string

// pickFormatter chooses a formatter based on the field's static type and the
// optional `table_format` struct tag.
func pickFormatter(ft reflect.Type, formatTag string) formatter {
	if ft == reflect.TypeFor[time.Time]() {
		return formatTime
	}

	switch formatTag {
	case "bytes":
		return formatBytes
	}

	switch ft.Kind() {
	case reflect.String:
		return formatString
	case reflect.Bool:
		return formatBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return formatInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return formatUint
	case reflect.Float32, reflect.Float64:
		return formatFloat
	case reflect.Slice, reflect.Array:
		return formatSlice
	}

	return formatDefault
}

func formatTime(v reflect.Value) string {
	t := v.Interface().(time.Time)
	if t.IsZero() {
		return "-"
	}
	return humanize.Time(t)
}

func formatBytes(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := v.Int()
		if n < 0 {
			return fmt.Sprint(n)
		}
		return humanize.IBytes(uint64(n))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return humanize.IBytes(v.Uint())
	}
	return formatDefault(v)
}

func formatString(v reflect.Value) string {
	s := v.String()
	if s == "" {
		return "-"
	}
	return s
}

func formatBool(v reflect.Value) string {
	if v.Bool() {
		return "true"
	}
	return "false"
}

func formatInt(v reflect.Value) string  { return fmt.Sprint(v.Int()) }
func formatUint(v reflect.Value) string { return fmt.Sprint(v.Uint()) }

func formatFloat(v reflect.Value) string {
	return fmt.Sprintf("%g", v.Float())
}

func formatSlice(v reflect.Value) string {
	if v.IsNil() || v.Len() == 0 {
		return "-"
	}
	return fmt.Sprintf("%d items", v.Len())
}

func formatDefault(v reflect.Value) string {
	if !v.IsValid() {
		return "-"
	}
	return fmt.Sprint(v.Interface())
}
