package cache

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
)

type fieldInfo struct {
	name     string
	isDive   bool
	optional bool
}

// parseTagsFast parses cache tags more efficiently
func parseTagsFast(cacheTag, jsonTag, fieldName string, fieldKind reflect.Kind) fieldInfo {
	info := fieldInfo{
		name: fieldName,
	}

	// Get field name from json tag or use lowercase field name
	if jsonName := strings.SplitN(jsonTag, ",", 2)[0]; jsonName != "" && jsonName != "-" {
		info.name = jsonName
	} else {
		info.name = strings.ToLower(fieldName)
	}

	if cacheTag == "" {
		// If no cache tag, struct fields should dive by default
		if fieldKind == reflect.Struct {
			info.isDive = true
		}
		return info
	}

	// Parse cache tag efficiently
	if fieldKind == reflect.Struct {
		info.isDive = !strings.HasPrefix(cacheTag, "nodive")
		if strings.Contains(cacheTag, "optional") {
			info.optional = true
		}
	} else {
		info.optional = strings.Contains(cacheTag, "optional")
	}

	return info
}

// isZeroValue checks if a value is zero without using reflect.DeepEqual
func isZeroValue(v reflect.Value, t reflect.Type) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		return v.IsZero() // Available in Go 1.13+
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(t).Interface())
	}
}

func getKey(data interface{}) (string, error) {
	if data == nil {
		return "", nil
	}

	v := reflect.ValueOf(data)

	// Handle pointer dereference once
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("expected struct, got %v", v.Kind())
	}

	t := v.Type()
	var keyBuilder strings.Builder

	numFields := v.NumField()
	for i := 0; i < numFields; i++ {
		field := v.Field(i)
		fieldT := t.Field(i)

		// Skip unexported fields
		if !field.CanInterface() {
			continue
		}

		// Parse tags efficiently
		info := parseTagsFast(
			fieldT.Tag.Get("cache"),
			fieldT.Tag.Get("json"),
			fieldT.Name,
			field.Kind(),
		)

		// Include fields that have cache attribute defined (even if empty) OR are dive structs
		_, hasCacheTag := fieldT.Tag.Lookup("cache")
		shouldInclude := hasCacheTag || info.isDive
		if !shouldInclude {
			continue
		}

		// Handle pointer dereference
		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				if !info.optional && !info.isDive {
					return "", fmt.Errorf("redis key: data cannot be empty: %v (use optional tag to allow empty value)", info.name)
				}
				continue
			}
			field = field.Elem()
		}

		// Check for zero values efficiently - but exclude bool false as it's a valid value
		if isZeroValue(field, field.Type()) {
			// For boolean fields, false is a valid value, not empty
			if field.Kind() == reflect.Bool {
				// Don't skip boolean false values
			} else if info.isDive || info.optional {
				continue
			} else {
				return "", fmt.Errorf("redis key: data cannot be empty: %v (use optional tag to allow empty value)", info.name)
			}
		}

		var value interface{}

		// Handle nested struct
		if info.isDive {
			nestedKey, err := getKey(field.Interface())
			if err != nil {
				return "", err
			}
			value = nestedKey
		} else {
			value = field.Interface()
		}

		// Build key efficiently
		keyBuilder.WriteString("#")
		keyBuilder.WriteString(info.name)
		keyBuilder.WriteString(":")
		keyBuilder.WriteString(fmt.Sprintf("%v", value))
	}

	return keyBuilder.String(), nil
}

func key(serviceName string, data interface{}, prefixes ...string) (string, error) {
	var keyBuilder strings.Builder
	keyBuilder.WriteString(serviceName)

	// Handle nil data case
	if data == nil {
		for _, p := range prefixes {
			keyBuilder.WriteString("#")
			keyBuilder.WriteString(p)
		}
		return keyBuilder.String(), nil
	}

	v := reflect.ValueOf(data)

	// Check for zero struct
	if isZeroValue(v, v.Type()) {
		log.Println("redis key: data struct is empty")
	}

	// Handle pointer dereference
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", fmt.Errorf("redis key: data pointer is nil")
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("redis key: data should be a struct")
	}

	keyBuilder.WriteString("#")
	keyBuilder.WriteString(v.Type().Name())

	// Add prefixes efficiently
	for _, p := range prefixes {
		keyBuilder.WriteString("#")
		keyBuilder.WriteString(p)
	}

	dataKey, err := getKey(data)
	if err != nil {
		return "", err
	}
	keyBuilder.WriteString(dataKey)

	return keyBuilder.String(), nil
}

// Key params
// @data: interface{}
// @prefixes: ...string
// return string, error
func Key(data interface{}, prefixes ...string) string {
	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		log.Println("redis key: SERVICE_NAME env variable should not be empty")
		return ""
	}

	key, err := key(serviceName, data, prefixes...)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	return key
}

// ExternalKey params
// @serviceName: string
// @data: interface{}
// @prefixes: ...string
// return string, error
func ExternalKey(serviceName string, data interface{}, prefixes ...string) string {
	key, err := key(serviceName, data, prefixes...)
	if err != nil {
		log.Println(err.Error())
		return ""
	}
	return key
}
