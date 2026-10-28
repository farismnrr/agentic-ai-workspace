package runner

import (
	"os"
	"reflect"
	"sort"
)

func sameMetadata(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) &&
		before.Mode() == after.Mode() && before.Size() == after.Size() &&
		before.ModTime().Equal(after.ModTime()) && sameChangeTime(before, after)
}

func sameChangeTime(before, after os.FileInfo) bool {
	first, firstFound := changeTime(before.Sys())
	second, secondFound := changeTime(after.Sys())
	return !firstFound || !secondFound || reflect.DeepEqual(first, second)
}

func changeTime(raw any) (any, bool) {
	value := reflect.ValueOf(raw)
	if !value.IsValid() {
		return nil, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, false
		}
		value = value.Elem()
	}
	for _, name := range []string{"Ctim", "Ctimespec"} {
		field := value.FieldByName(name)
		if field.IsValid() && field.CanInterface() {
			return field.Interface(), true
		}
	}
	return nil, false
}

func changedEffects(before, after effectSnapshot) ([]string, bool) {
	if !before.complete || !after.complete {
		return nil, false
	}
	paths := make([]string, 0)
	for name, current := range after.files {
		previous, exists := before.files[name]
		if !exists || current.digest != previous.digest || !sameMetadata(previous.info, current.info) {
			paths = append(paths, name)
		}
	}
	for name := range before.files {
		if _, exists := after.files[name]; !exists {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	return paths, true
}
