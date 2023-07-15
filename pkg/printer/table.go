// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// PrintTable prints the data to out in a table format.
// If data is a simple type (bool, int, string, etc), it will print it as-is.
// If data is a slice/map, it will print a summary table.
// If data is a struct, it will print simple fields as 'key: value' table and complex fields as sub-tables.
// Non-printable types will return an error.
//
// For structs, some field tags are supported:
// - if "name" tag exists, that will be used instead of the field name.
// - if "priority" tag exists, it will be used to sort the fields. Higher priority will be printed first.
func PrintTable(out io.Writer, data any) (err error) {
	w := tabwriter.NewWriter(out, 2, 2, 2, ' ', 0)
	defer w.Flush()
	return printTable(w, data)
}

func printTable(out io.Writer, data any) (err error) {
	if data == nil {
		return nil
	}

	kind := reflect.TypeOf(data).Kind()
	switch kind {
	case reflect.Pointer: // just dereference it:
		return printTable(out, reflect.ValueOf(data).Elem().Interface())
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		_, err = fmt.Fprintln(out, data)
	case reflect.Array, reflect.Slice,
		reflect.Map:
		err = printTableList(out, data)
	case reflect.Struct:
		printTableStruct(out, data)
	case reflect.Invalid, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		err = fmt.Errorf("cannot print type %T (kind: %s)", data, kind.String())
	default:
		err = fmt.Errorf("unknown type for printing: %T (kind: %s)", data, kind.String())
	}

	return err
}

func printTableStruct(out io.Writer, data any) {
	o := &StructuredOutput{}
	keys := GetSortedStructFields(reflect.TypeOf(data))
	for _, key := range keys {
		o.ProcessStructField(key.printName, reflect.ValueOf(data).FieldByName(key.fieldName))
	}
	o.PrintTo(out)
}

func printTableList(out io.Writer, data any) (err error) {
	value := reflect.ValueOf(data)
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		return printTableListOfSlice(out, data)
	case reflect.Map:
		if isMapOfSimple(data) {
			mapKeys := reflect.ValueOf(data).MapKeys()
			sort.Slice(mapKeys, func(i, j int) bool { return fmt.Sprint(mapKeys[i].Interface()) < fmt.Sprint(mapKeys[j].Interface()) })
			for i, k := range mapKeys {
				if i > 0 {
					fmt.Fprint(out, ", ")
				}
				fmt.Fprintf(out, "%v: %v", k.Interface(), reflect.ValueOf(data).MapIndex(k).Interface())
			}
			fmt.Fprintln(out, "")
			return nil
		}
		return printTableListOfMap(out, data)
	default:
		return fmt.Errorf("cannot print type as list: %T (%s)", data, value.Kind().String())
	}
}

type OutputField struct {
	name  string
	value string
}
type StructuredOutput struct {
	simpleData  []OutputField
	complexData []OutputField
}

// PrintTo will output the structured output to the given io.Writer.
// eg: PrintTo(os.Stdout) will output:
// simpleField1:\tvalue1
// simpleField2:\tvalue2
//
// complexField1:
// complexValue1...
//
// complexField2:
// complexValue2...
func (o *StructuredOutput) PrintTo(output io.Writer) {
	for _, f := range o.simpleData {
		fmt.Fprintf(output, "%s:\t%s\n", normalizeName(f.name), f.value)
	}
	for _, f := range o.complexData {
		fmt.Fprintf(output, "\n%s:\n%s", normalizeName(f.name), f.value)
		if !strings.HasSuffix(f.value, "\n") {
			fmt.Fprintln(output)
		}
	}
}

// ProcessStructField will process a single field of a struct.
// It will differentiate between simple and complex fields and process them accordingly.
// Use PrintTo() to output the processed data.
func (o *StructuredOutput) ProcessStructField(name string, value reflect.Value) error {
	if o.simpleData == nil {
		o.simpleData = []OutputField{}
	}
	if o.complexData == nil {
		o.complexData = []OutputField{}
	}

	kind := value.Kind()
	switch kind {
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return o.ProcessStructField(name, value.Elem())
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		o.simpleData = append(o.simpleData, OutputField{name: name, value: ParseField(value)})
	case reflect.Array, reflect.Slice,
		reflect.Map:
		if value.Len() == 0 {
			return nil
		}
		if isCollectionOfSimple(value.Interface()) {
			o.simpleData = append(o.simpleData, OutputField{name: name, value: ParseField(value)})
		} else {
			o.complexData = append(o.complexData, OutputField{name: name, value: ParseField(value)})
		}
	case reflect.Struct:
		o.complexData = append(o.complexData, OutputField{name: name, value: ParseField(value)})
	default:
		return fmt.Errorf("cannot process field %q of type %T (kind: %s)", name, value.Interface(), kind.String())
	}
	return nil
}

// ParseField will parse a single value into a string.
// If value is a simple type (bool, int, string, etc), it will return it as-is.
// If value is a slice/map of simple types, it will return a comma-separated list.
// If value is a slice/map of complex types, it will return a summary table.
// If value is a struct, it will return a summary table.
func ParseField(value reflect.Value) string {
	kind := value.Kind()
	switch kind {
	case reflect.Pointer:
		return ParseField(value.Elem())
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		return fmt.Sprint(value.Interface())
	case reflect.Array, reflect.Slice:
		return parseStructFieldAsSubList(value)
	case reflect.Map:
		if isCollectionOfSimple(value.Interface()) {
			mapKeys := value.MapKeys()
			sort.Slice(mapKeys, func(i, j int) bool { return fmt.Sprint(mapKeys[i].Interface()) < fmt.Sprint(mapKeys[j].Interface()) })
			buf := &strings.Builder{}
			for i, k := range mapKeys {
				if i > 0 {
					fmt.Fprint(buf, ", ")
				}
				fmt.Fprintf(buf, "%v: %v", k.Interface(), value.MapIndex(k).Interface())
			}
			return buf.String()
		}
		return parseStructFieldAsSubList(value)
	case reflect.Struct:
		return parseStructFieldAsSubStruct(value)

	default:
		return fmt.Sprintf("<not parseable (kind: %s)>", kind.String())
	}
}

func parseStructFieldAsSubList(value reflect.Value) string {
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return ""
	}
	if value.Len() == 0 {
		return ""
	}

	if isCollectionOfSimple(value.Interface()) {
		buf := &strings.Builder{}
		for i := 0; i < value.Len(); i++ {
			if i > 0 {
				fmt.Fprint(buf, ", ")
			}
			fmt.Fprintf(buf, "%v", value.Index(i).Interface())
		}
		return buf.String()
	}

	sliceElement := value.Type().Elem()
	if sliceElement.Kind() == reflect.Pointer {
		sliceElement = sliceElement.Elem()
	}

	switch sliceElement.Kind() {
	case reflect.Array, reflect.Slice:
		if isCollectionOfSimple(value.Index(0).Interface()) {
			buf := &strings.Builder{}
			for i := 0; i < value.Len(); i++ {
				if i > 0 {
					fmt.Fprint(buf, ", ")
				}
				fmt.Fprintf(buf, "%v", value.Index(i).Interface())
			}
			return buf.String()
		}
		return fmt.Sprintf("[]%s{...}", sliceElement.String())
	case reflect.Map:
		buf := &strings.Builder{}
		for i := 0; i < value.Len(); i++ {
			if i > 0 {
				fmt.Fprint(buf, "\n")
			}
			item := value.Index(i)
			if item.Kind() == reflect.Pointer {
				item = item.Elem()
			}
			keys := item.MapKeys()
			sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface()) })
			for j, k := range keys {
				if j == 0 {
					fmt.Fprintf(buf, "map{")
				} else {
					fmt.Fprint(buf, ", ")
				}
				fmt.Fprintf(buf, "%v", k.Interface())
			}
			fmt.Fprint(buf, "}")
		}
		return buf.String()
	case reflect.Struct:
		keys := GetSortedStructFields(sliceElement)

		buf := &strings.Builder{}
		for i, k := range keys {
			if i > 0 {
				fmt.Fprint(buf, "\t")
			}
			fmt.Fprintf(buf, "%s", strings.ToUpper(normalizeName(k.printName)))
		}

		for i := 0; i < value.Len(); i++ {
			fmt.Fprint(buf, "\n")
			for j, k := range keys {
				if j > 0 {
					fmt.Fprint(buf, "\t")
				}
				item := value.Index(i)
				if item.Kind() == reflect.Pointer {
					item = item.Elem()
				}

				field := item.FieldByName(k.fieldName)
				if field.Kind() == reflect.Pointer {
					field = field.Elem()
				}
				fmt.Fprintf(buf, "%v", field.Interface())
			}
		}
		fmt.Fprintln(buf)
		return buf.String()
	default:
		return fmt.Sprintf("[]%s{...}", sliceElement.String())
	}
}

func parseStructFieldAsSubStruct(value reflect.Value) string {
	if value.Kind() != reflect.Struct {
		return ""
	}

	keys := GetSortedStructFields(value.Type())

	buf := &strings.Builder{}
	for i, k := range keys {
		if i > 0 {
			fmt.Fprint(buf, "\t")
		}
		fmt.Fprintf(buf, "%s", strings.ToUpper(normalizeName(k.printName)))
	}
	fmt.Fprintln(buf)

	for i, k := range keys {
		if i > 0 {
			fmt.Fprint(buf, "\t")
		}
		item := value.FieldByName(k.fieldName).Interface()
		fmt.Fprintf(buf, "%v", item)
	}
	fmt.Fprintln(buf)

	return buf.String()
}

func printTableListOfSlice(out io.Writer, data any) (err error) {
	value := reflect.ValueOf(data)
	if value.Len() == 0 {
		return nil
	}

	subKind := value.Type().Elem().Kind()
	if subKind == reflect.Pointer {
		subKind = value.Type().Elem().Elem().Kind()
	}

	if subKind == reflect.Struct {
		fmt.Fprintln(out, ParseField(value))
		return nil
	}

	for i := 0; i < value.Len(); i++ {
		fmt.Fprintln(out, ParseField(value.Index(i)))
	}

	return nil
}
func printTableListOfMap(out io.Writer, data any) (err error) {
	return fmt.Errorf("printTableListOfMap not implemented")
}

func isCollectionOfSimple(data any) bool {
	return isSliceOfSimple(data) || isMapOfSimple(data)
}

func isSliceOfSimple(data any) bool {
	kind := reflect.TypeOf(data).Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return false
	}

	sliceKind := reflect.TypeOf(data).Elem().Kind()

	switch sliceKind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		return true
	default:
		return false
	}
}

func isMapOfSimple(data any) bool {
	if reflect.TypeOf(data).Kind() != reflect.Map {
		return false
	}

	valueKind := reflect.TypeOf(data).Elem().Kind()
	keyKind := reflect.TypeOf(data).Key().Kind()
	isKeyKindSimple, isValueKindSimple := false, false

	switch keyKind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		isKeyKindSimple = true
	}
	switch valueKind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.String:
		isValueKindSimple = true
	}

	return isKeyKindSimple && isValueKindSimple
}

type structFieldsSortable struct {
	fieldName string
	printName string
	priority  int
}

func GetSortedStructFields(structType reflect.Type) []structFieldsSortable {
	if structType.Kind() != reflect.Struct {
		return nil
	}

	fields := []structFieldsSortable{}
	for _, field := range reflect.VisibleFields(structType) {
		if !field.IsExported() {
			continue
		}

		priorityStr := field.Tag.Get("priority")
		priority, _ := strconv.Atoi(priorityStr)

		printName := field.Name
		if tag := field.Tag.Get("name"); tag != "" {
			printName = tag
		}
		fields = append(fields, structFieldsSortable{priority: priority, fieldName: field.Name, printName: printName})
	}
	sort.Slice(fields, func(i, j int) bool {
		if fields[i].priority == fields[j].priority {
			return fields[i].printName < fields[j].printName
		}
		return fields[i].priority > fields[j].priority
	})

	return fields
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// normalizeName will normalize a string to be used as a field name.
// Meaning, it will convert CamelCase to Title Case.
func normalizeName(s string) string {
	ret := matchFirstCap.ReplaceAllString(s, "${1} ${2}")
	ret = matchAllCap.ReplaceAllString(ret, "${1} ${2}")
	return strings.Title(ret) //lint:ignore SA1019 // structure fields are safe
}
