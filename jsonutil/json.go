package jsonutil

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/goccy/go-json"
)

func UnmarshalGraphQL(data []byte, out any) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("cannot decode into non-pointer %T", out)
	}

	m := map[string]any{}
	err := json.Unmarshal(data, m)
	if err != nil {
		return err
	}

	err = ConvertMapToStruct(m, rv)
	if err != nil {
		return fmt.Errorf("convert to struct: %w", err)
	}

	return nil
}

func ConvertMapToStruct(m map[string]any, s reflect.Value) error {
	stValue := s
	if stValue.Kind() == reflect.Pointer {
		stValue = stValue.Elem()
	}
	sType := stValue.Type()
	if sType.Kind() == reflect.Interface {
		return nil
	}
	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)
		fieldName := field.Name
		if jsonTag, ok := field.Tag.Lookup("json"); ok {
			jsonTagValues := strings.Split(jsonTag, ",")
			fieldName = jsonTagValues[0]
		}
		if value, ok := m[fieldName]; ok {
			v := reflect.ValueOf(value)

			switch v.Kind() {
			case reflect.Map:
				newStruct := createStruct(field.Type)
				if !newStruct.IsValid() {
					continue
				}
				err := ConvertMapToStruct(value.(map[string]any), newStruct)
				if err != nil {
					return err
				}
				if field.Type.Kind() != reflect.Pointer {
					newStruct = newStruct.Elem()
				}
				stValue.Field(i).Set(newStruct)

			case reflect.Slice:
				s, err := ConvertSlice(v, field.Type)
				if err != nil {
					return err
				}
				if field.Type.Kind() == reflect.Pointer {
					s = s.Elem()
				}
				stValue.Field(i).Set(s)

			default:
				fieldT := field.Type
				if field.Type.Kind() == reflect.Pointer {
					fieldT = fieldT.Elem()
				}
				convertedV, err := convertValue(v, fieldT)
				if err != nil {
					return fmt.Errorf("can not assign value `%v` of type %v to field %s of type %v", value, v.Kind(), field.Name, field.Type)
				}
				if field.Type.Kind() == reflect.Pointer {
					stValue.Field(i).Set(reflect.New(fieldT))
					stValue.Field(i).Elem().Set(convertedV)
				} else {
					stValue.Field(i).Set(convertedV)
				}
			}
		}
	}
	return nil
}

func ConvertSlice(v reflect.Value, t reflect.Type) (reflect.Value, error) {
	newSlice := reflect.New(t).Elem()
	for i := 0; i < v.Len(); i++ {
		var sValue reflect.Value
		switch v.Index(i).Kind() {
		case reflect.Map:
			sValue = createStruct(t.Elem())
			if !sValue.IsValid() {
				continue
			}
			err := ConvertMapToStruct(v.Index(i).Interface().(map[string]any), sValue)
			if err != nil {
				return reflect.Value{}, err
			}
		default:
			sValue = v.Index(i)
		}
		newSlice = reflect.Append(newSlice, sValue)
	}
	return newSlice, nil
}

func convertValue(value reflect.Value, t reflect.Type) (result reflect.Value, err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	if value.Kind() == reflect.String && t.Kind() == reflect.Struct {
		var v time.Time
		v, err = time.Parse(time.RFC3339, value.String())
		if err != nil {
			return
		}
		result = reflect.ValueOf(v)
		return
	}
	result = value.Convert(t)
	return
}

func createStruct(sType reflect.Type) reflect.Value {
	nStruct := reflect.New(sType)
	nStructT := sType
	if sType.Kind() == reflect.Pointer {
		nStruct = reflect.New(sType.Elem())
		nStructT = sType.Elem()
	}
	if nStructT.Kind() == reflect.Interface {
		return reflect.Value{}
	}
	initializeStruct(nStruct.Elem(), nStructT)
	return nStruct
}

func initializeStruct(v reflect.Value, t reflect.Type) {
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		ft := t.Field(i)
		switch ft.Type.Kind() {
		case reflect.Map:
			f.Set(reflect.MakeMap(ft.Type))
		case reflect.Slice:
			f.Set(reflect.MakeSlice(ft.Type, 0, 0))
		case reflect.Chan:
			f.Set(reflect.MakeChan(ft.Type, 0))
		case reflect.Struct:
			initializeStruct(f, ft.Type)
		case reflect.Ptr:
			if ft.Type.Elem().Kind() == reflect.Struct {
				fv := reflect.New(ft.Type.Elem())
				initializeStruct(fv.Elem(), ft.Type.Elem())
				f.Set(fv)
			}
		default:
		}
	}
}
