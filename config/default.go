package config

import (
	"reflect"
	"strconv"
)

func configDefault(c any) {
	configDefaultReflect(reflect.ValueOf(c).Elem())
}

func configDefaultReflect(v reflect.Value) {
	tp := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}

		tag := tp.Field(i).Tag.Get("default")
		if tag == "" {
			continue
		}

		if f.Kind() == reflect.Struct {
			configDefaultReflect(f)
			continue
		}

		setFuncs(f, tag)
	}
}

func setFuncs(v reflect.Value, s string) {
	switch v.Type().Kind() {
	case reflect.Int, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			panic(err)
		}
		v.SetInt(i)
	case reflect.String:
		v.SetString(s)
	}

	panic("not support type: " + v.Type().String())
}
