package configutil

import (
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/api7/a7/pkg/api"
)

// envVarRegex matches an un-escaped `${VAR}` reference.
// It deliberately does not match `\${VAR}` because the escape is handled
// by a pre-pass that swaps escaped references for a placeholder.
var envVarRegex = regexp.MustCompile(`\$\{(\w+)\}`)

// escapedVarRegex matches `\${VAR}` so we can mark escaped references
// before substitution and restore them as literal `${VAR}` afterwards.
var escapedVarRegex = regexp.MustCompile(`\\\$\{(\w+)\}`)

const escapedEnvPlaceholder = "\x00__A7_ESCAPED_ENV_VAR__\x00"

// substituteEnvString applies `${VAR}` substitution to a single string.
// Rules:
//   - `${VAR}` is replaced with os.Getenv("VAR").
//   - If VAR is unset, the literal `${VAR}` is preserved unchanged.
//   - `\${VAR}` becomes a literal `${VAR}` (the backslash is consumed).
//   - Substitution is applied left-to-right; the replacement text itself
//     is NOT re-scanned for further references.
func substituteEnvString(value string) string {
	if !strings.ContainsAny(value, "$\\") {
		return value
	}

	// Step 1: stash escaped sequences so they survive the substitution pass.
	stashed := escapedVarRegex.ReplaceAllStringFunc(value, func(match string) string {
		name := escapedVarRegex.FindStringSubmatch(match)[1]
		return escapedEnvPlaceholder + name + escapedEnvPlaceholder
	})

	// Step 2: substitute real references. If the variable is unset, keep the
	// literal `${VAR}` intact rather than replacing it with an empty string.
	substituted := envVarRegex.ReplaceAllStringFunc(stashed, func(match string) string {
		name := envVarRegex.FindStringSubmatch(match)[1]
		v, ok := os.LookupEnv(name)
		if !ok {
			return match
		}
		return v
	})

	// Step 3: restore escaped sequences as literal `${VAR}`.
	if !strings.Contains(substituted, escapedEnvPlaceholder) {
		return substituted
	}
	placeholderRegex := regexp.MustCompile(regexp.QuoteMeta(escapedEnvPlaceholder) + `(\w+)` + regexp.QuoteMeta(escapedEnvPlaceholder))
	return placeholderRegex.ReplaceAllStringFunc(substituted, func(match string) string {
		name := placeholderRegex.FindStringSubmatch(match)[1]
		return "${" + name + "}"
	})
}

// applyEnvSubstitution walks every string value inside the parsed ConfigFile
// and applies `${VAR}` substitution per substituteEnvString. Numbers, bools,
// and nulls are left untouched. Map keys are NOT substituted, matching the
// adc reference behaviour (see "object key contains variables will not be
// parsed" in adc's tests).
func applyEnvSubstitution(cfg *api.ConfigFile) error {
	if cfg == nil {
		return nil
	}
	walkValue(reflect.ValueOf(cfg).Elem())
	return nil
}

// walkValue mutates reflect-addressable string values in place.
func walkValue(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return
		}
		// For interface values we cannot set fields of the underlying value
		// directly; rebuild via the generic any-walker so the holding container
		// can replace it.
		if v.Kind() == reflect.Interface {
			newVal := walkAny(v.Interface())
			if v.CanSet() {
				v.Set(reflect.ValueOf(newVal))
			}
			return
		}
		walkValue(v.Elem())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if !v.Type().Field(i).IsExported() {
				continue
			}
			walkValue(f)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkValue(v.Index(i))
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			val := iter.Value()
			// Map values are not addressable; build a replacement and
			// reassign it under the same key.
			newVal := walkAny(val.Interface())
			v.SetMapIndex(k, reflect.ValueOf(newVal))
		}
	case reflect.String:
		if !v.CanSet() {
			return
		}
		v.SetString(substituteEnvString(v.String()))
	}
}

// walkAny rebuilds an arbitrary value (typically from a map[string]interface{})
// with string substitution applied. It is used for paths the reflect-in-place
// walker cannot mutate directly (map values, interface holders).
func walkAny(value interface{}) interface{} {
	switch typed := value.(type) {
	case string:
		return substituteEnvString(typed)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for k, vv := range typed {
			out[k] = walkAny(vv)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, vv := range typed {
			out[i] = walkAny(vv)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(typed))
		for k, vv := range typed {
			out[k] = substituteEnvString(vv)
		}
		return out
	case []string:
		out := make([]string, len(typed))
		for i, vv := range typed {
			out[i] = substituteEnvString(vv)
		}
		return out
	case nil:
		return nil
	default:
		// For other concrete types (numbers, bools, custom maps/structs) reuse
		// the reflect-based walker so nested strings still get substituted.
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			return value
		}
		// We need an addressable copy to mutate fields/elements.
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		walkValue(ptr.Elem())
		return ptr.Elem().Interface()
	}
}
