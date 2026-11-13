package assert

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// форматирует необязательное пояснение к проверке
func msg(args []any) string {
	if len(args) == 0 {
		return ""
	}
	if format, ok := args[0].(string); ok {
		return " " + fmt.Sprintf(format, args[1:]...)
	}
	return " " + fmt.Sprint(args...)
}

// проверяет, что ошибки нет (err == nil)
func NoError(t *testing.T, err error, args ...any) bool {
	t.Helper()
	if err != nil {
		t.Errorf("ожидалось отсутствие ошибки, получено: %v.%s", err, msg(args))
		return false
	}
	return true
}

// проверяет, что ошибка есть (err != nil)
func Error(t *testing.T, err error, args ...any) bool {
	t.Helper()
	if err == nil {
		t.Errorf("ожидалась ошибка, получено nil.%s", msg(args))
		return false
	}
	return true
}

// проверяет, что err оборачивает target (errors.Is)
func ErrorIs(t *testing.T, err, target error, args ...any) bool {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("ожидалась ошибка, оборачивающая %v, получено: %v.%s", target, err, msg(args))
		return false
	}
	return true
}

// сравнивает ожидаемое и фактическое значения через reflect.DeepEqual
func Equal(t *testing.T, expected, actual any, args ...any) bool {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("ожидалось %#v, получено %#v.%s", expected, actual, msg(args))
		return false
	}
	return true
}

// проверяет, что условие истинно
func True(t *testing.T, value bool, args ...any) bool {
	t.Helper()
	if !value {
		t.Errorf("ожидалось true, получено false.%s", msg(args))
		return false
	}
	return true
}

// проверяет длину среза, массива, карты, строки или канала
func Len(t *testing.T, object any, length int, args ...any) bool {
	t.Helper()
	v := reflect.ValueOf(object)
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String, reflect.Chan:
		if v.Len() != length {
			t.Errorf("ожидалась длина %d, получено %d.%s", length, v.Len(), msg(args))
			return false
		}
		return true
	default:
		t.Errorf("Len: тип %T не имеет длины.%s", object, msg(args))
		return false
	}
}
