package utils

import (
	"errors"
	"reflect"
	"strconv"
)

var ErrNotPointerToStruct error = errors.New("s must be a pointer to a struct")

func MapDefaults(s interface{}) error {
	val := reflect.ValueOf(s)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return ErrNotPointerToStruct
	}

	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if field.CanSet() && field.IsZero() {
			if defaultValue, ok := fieldType.Tag.Lookup("default"); ok {
				if defaultValue == "" {
					continue
				}

				switch field.Kind() {
				case reflect.String:
					field.SetString(defaultValue)
				case reflect.Int:
					if intValue, err := strconv.ParseInt(defaultValue, 10, 64); err == nil {
						field.SetInt(intValue)
					} else {
						return err
					}
				case reflect.Bool:
					if boolValue, err := strconv.ParseBool(defaultValue); err == nil {
						field.SetBool(boolValue)
					} else {
						return err
					}
				default:
					panic("unsupported type")
				}
			}
		}
	}
	return nil
}
