package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// Map is a map containing configuration values.
type Map map[string]string

// Parse reads the config
func Parse[T any](configMap Map) (T, error) {
	var target T
	targetType := reflect.TypeOf(target)
	switch targetType.Kind() {
	case reflect.Struct:
		err := ParseInto(configMap, &target)
		return target, err
	case reflect.Pointer:
		// TODO: Handle cases other than pointer to struct.
		p := reflect.ValueOf(&target).Elem()
		if p.IsNil() {
			p.Set(reflect.New(targetType.Elem()))
		}
		err := ParseInto(configMap, p.Interface())
		return target, err
	default:
		return target, fmt.Errorf("unsupported target type \"%T\"", target)
	}
}

func ParseInto(configMap map[string]string, target any) error {
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Pointer {
		return fmt.Errorf("unsupported target type \"%qT\"", target)
	}

	targetType = targetType.Elem()
	if targetType.Kind() != reflect.Struct {
		return fmt.Errorf("unsupported target type \"%T\"", target)
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.IsNil() {
		return errors.New("target was nil")
	}
	targetValue = targetValue.Elem()

	for i := 0; i < targetType.NumField(); i++ {
		fieldInfo := targetType.Field(i)
		if !fieldInfo.IsExported() {
			continue
		}
		tag, ok := fieldInfo.Tag.Lookup("config_key")
		if !ok {
			continue
		}
		configKey := tag
		configVal, ok := configMap[tag]
		if !ok {
			continue
		}
		field := targetValue.Field(i)

		switch field.Kind() {
		case reflect.String:
			field.SetString(configVal)
		case reflect.Bool:
			v, err := strconv.ParseBool(configVal)
			if err != nil {
				return fmt.Errorf("parse %q=%q: %v", configKey, configVal, err)
			}
			field.SetBool(v)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bitSize := int(field.Type().Size() * 8)
			v, err := strconv.ParseInt(configVal, 10, bitSize)
			if err != nil {
				return fmt.Errorf("parse %q=%q: %v", configKey, configVal, err)
			}
			field.SetInt(v)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			bitSize := int(field.Type().Size() * 8)
			v, err := strconv.ParseUint(configVal, 10, bitSize)
			if err != nil {
				return fmt.Errorf("parse %q=%q: %v", configKey, configVal, err)
			}
			field.SetUint(v)
		case reflect.Float32, reflect.Float64:
			bitSize := int(field.Type().Size() * 8)
			v, err := strconv.ParseFloat(configVal, bitSize)
			if err != nil {
				return fmt.Errorf("parse %q=%q: %v", configKey, configVal, err)
			}
			field.SetFloat(v)
		}
	}
	return nil
}
