// Copyright © 2023 tsuru-client authors
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package printer

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTable(t *testing.T) {
	t.Parallel()
	t.Run("simple_cases", func(t *testing.T) {
		for i, tc := range []struct {
			in       any
			expected string
		}{
			{in: "", expected: ""},
			{in: "string", expected: "string"},
			{in: 1, expected: "1"},
			{in: true, expected: "true"},
			{in: 1.1, expected: "1.1"},
			{in: []string{"slice1", "slice2"}, expected: "slice1\nslice2"},
			{in: []int{1, 2}, expected: "1\n2"},
			{in: map[string]int{"key1": 1, "key2": 2}, expected: "key1: 1, key2: 2"},
			{in: map[int]float32{42: 9.9, -35: 2.999}, expected: "-35: 2.999, 42: 9.9"},
			{in: map[string]string{"key1": "value1", "key2": "value2"}, expected: "key1: value1, key2: value2"},
		} {
			buf := &bytes.Buffer{}
			err := PrintTable(buf, tc.in)
			assert.NoError(t, err, fmt.Sprintf("during test case %d: %+v", i, tc.in))
			assert.Equal(t, tc.expected+"\n", buf.String(), fmt.Sprintf("during test case %d: %+v", i, tc.in))
		}

		// nil case
		buf := &bytes.Buffer{}
		err := PrintTable(buf, nil)
		assert.NoError(t, err)
		assert.Equal(t, "", buf.String())
	})

	t.Run("error_cases", func(t *testing.T) {
		for _, tc := range []struct {
			in       any
			expected string
		}{
			{in: make(chan int), expected: "cannot print type chan int (kind: chan)"},
			{in: func() {}, expected: "cannot print type func() (kind: func)"},
		} {
			buf := &bytes.Buffer{}
			err := PrintTable(buf, tc.in)
			assert.Error(t, err)
			assert.ErrorContains(t, err, tc.expected)
		}
	})

	t.Run("ObjectInfo", func(t *testing.T) {
		data := struct {
			FieldStr     string `priority:"1"`
			FieldInt     int
			FieldStruct1 struct {
				A int
				B int
				C int `priority:"1"`
			}
			FieldStruct2 struct {
				A int `priority:"-1"`
				B int
				C int `priority:"1"`
			} `priority:"1"`
			FieldBool bool
		}{FieldStr: "string"}

		expected := strings.TrimLeft(`
Field Str:   string
Field Bool:  false
Field Int:   0

Field Struct2:
C  B  A
0  0  0

Field Struct1:
C  A  B
0  0  0
`, "\n")

		buf := &bytes.Buffer{}
		err := PrintTable(buf, data)
		assert.NoError(t, err)
		assert.Equal(t, expected, buf.String())
	})

	t.Run("ObjectList", func(t *testing.T) {
		type myStructO struct {
			Fieldstr     string `priority:"1"`
			Fieldint     int
			Fieldstruct1 struct {
				A int
				B int
				C int `priority:"1"`
			}
			Fieldstruct2 struct {
				A int `priority:"-1"`
				B int
				C int `priority:"1"`
			} `priority:"1"`
			FieldBool bool
		}

		expected := strings.TrimLeft(`
FIELDSTR  FIELDSTRUCT2  FIELD BOOL  FIELDINT  FIELDSTRUCT1
string1   {0 0 0}       false       0         {0 0 0}
string2   {0 0 0}       false       0         {0 0 0}

`, "\n")

		for _, tc := range []struct {
			data     any
			expected string
		}{
			{data: []myStructO{}, expected: ""},
			{data: []*myStructO{}, expected: ""},
			{data: []myStructO{{Fieldstr: "string1"}, {Fieldstr: "string2"}}, expected: expected},
			{data: []*myStructO{{Fieldstr: "string1"}, {Fieldstr: "string2"}}, expected: expected},
		} {
			buf := &bytes.Buffer{}
			err := PrintTable(buf, tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, buf.String())
		}
	})
}

func TestStructuredOutput_String(t *testing.T) {
	t.Parallel()
	t.Run("simple_cases", func(t *testing.T) {
		for _, tc := range []struct {
			in       *StructuredOutput
			expected string
		}{
			{in: &StructuredOutput{
				simpleData: []OutputField{{"a1", "aaa"}, {"b", "bbb"}},
			}, expected: "A1:\taaa\nB:\tbbb\n"},
			{in: &StructuredOutput{
				complexData: []OutputField{{"x", "xxx"}, {"y", "yyy"}},
			}, expected: "\nX:\nxxx\n\nY:\nyyy\n"},
			{in: &StructuredOutput{
				simpleData:  []OutputField{{"a1", "aaa"}, {"b", "bbb"}},
				complexData: []OutputField{{"x", "xxx"}, {"y", "yyy"}},
			}, expected: "A1:\taaa\nB:\tbbb\n\nX:\nxxx\n\nY:\nyyy\n"},
		} {
			buf := &bytes.Buffer{}
			tc.in.PrintTo(buf)
			assert.Equal(t, tc.expected, buf.String())
		}
	})
}

func TestStructuredOutput_ProcessStructField(t *testing.T) {
	t.Parallel()
	newS := "onestring"
	type mystruct struct {
		Fieldstr string
	}

	t.Run("simpleData", func(t *testing.T) {
		for i, tc := range []struct {
			data     any
			expected string
		}{
			{data: "string", expected: "string"},
			{data: &newS, expected: "onestring"},
			{data: 42, expected: "42"},
			{data: true, expected: "true"},
			{data: []string{"a", "b", "c"}, expected: "a, b, c"},
			{data: &[]string{"a", "b", "c"}, expected: "a, b, c"},
			{data: []int{1, 2, 3}, expected: "1, 2, 3"},
			{data: []bool{true, false, true}, expected: "true, false, true"},
			{data: map[string]string{"a": "aaa", "b": "bbb"}, expected: "a: aaa, b: bbb"},
			{data: map[string]int{"a": 1, "b": 2}, expected: "a: 1, b: 2"},
			{data: map[int]int{1: 1, 2: 2}, expected: "1: 1, 2: 2"},
			{data: map[int]int{2: 1, 1: 2}, expected: "1: 2, 2: 1"}, // get them sorted
		} {
			o := &StructuredOutput{}
			v := reflect.ValueOf(tc.data)
			err := o.ProcessStructField("test", v)
			assert.NoError(t, err, fmt.Sprintf("during test case %d: %+v", i, tc.data))
			assert.Equal(t, tc.expected, o.simpleData[0].value, fmt.Sprintf("during test case %d: %+v", i, tc.data))
			assert.Equal(t, tc.expected, o.simpleData[0].value, fmt.Sprintf("during test case %d: %+v", i, tc.data))
			assert.Equal(t, 1, len(o.simpleData), fmt.Sprintf("during test case %d: %+v", i, tc.data))
			assert.Equal(t, 0, len(o.complexData), fmt.Sprintf("during test case %d: %+v", i, tc.data))
		}
	})

	t.Run("emptyOrNilData", func(t *testing.T) {
		var nilSlice []string
		var nilMap map[string]string
		var nilStructP *mystruct

		for i, tc := range []struct {
			data any
		}{
			{data: []string{}},
			{data: map[string]string{}},
			{data: []mystruct{}},
			{data: map[string]mystruct{}},
			{data: []*mystruct{}},
			{data: map[string]*mystruct{}},
			{data: map[string]*mystruct{}},
			{data: nilSlice},
			{data: nilMap},
			{data: nilStructP},
		} {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				o := &StructuredOutput{}
				v := reflect.ValueOf(tc.data)
				err := o.ProcessStructField("test", v)
				assert.NoError(t, err, fmt.Sprint("during test", i))
				assert.Equal(t, 0, len(o.simpleData), fmt.Sprint("during test", i))
				assert.Equal(t, 0, len(o.complexData), fmt.Sprint("during test", i))
			})
		}
	})

	t.Run("complexData", func(t *testing.T) {
		data := struct {
			FieldStr string
		}{FieldStr: "string"}

		o := &StructuredOutput{}
		v := reflect.ValueOf(data)
		err := o.ProcessStructField("test", v)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(o.simpleData))
		assert.Equal(t, 1, len(o.complexData))
	})
}

func TestStructuredOutput_processComplexField(t *testing.T) {
	t.Parallel()

	type struct1 struct {
		Fieldstr         string
		Fieldint         int
		nonExportedField string
	}
	type struct2 struct {
		Fieldstr         string `priority:"1"`
		Fieldint         int
		nonExportedField string
	}

	t.Run("structData", func(t *testing.T) {
		for _, tc := range []struct {
			data     any
			expected string
		}{
			{data: struct1{Fieldstr: "string", Fieldint: 42}, expected: "FIELDINT\tFIELDSTR\n42\tstring\n"},
			{data: &struct1{Fieldstr: "string", Fieldint: 42}, expected: "FIELDINT\tFIELDSTR\n42\tstring\n"},
			{data: struct1{nonExportedField: "hidden"}, expected: "FIELDINT\tFIELDSTR\n0\t\n"},
			{data: struct2{Fieldstr: "string", Fieldint: 42}, expected: "FIELDSTR\tFIELDINT\nstring\t42\n"},
			{data: &struct2{Fieldstr: "string", Fieldint: 42}, expected: "FIELDSTR\tFIELDINT\nstring\t42\n"},
			{data: struct2{nonExportedField: "hidden"}, expected: "FIELDSTR\tFIELDINT\n\t0\n"},
		} {
			v := reflect.ValueOf(tc.data)
			s := ParseField(v)
			assert.Equal(t, tc.expected, s)
		}
	})

	t.Run("listOfStructs", func(t *testing.T) {
		for i, tc := range []struct {
			data     any
			expected string
		}{
			{data: []struct1{}, expected: ""},
			{data: &[]struct1{}, expected: ""},
			{data: []struct1{
				{Fieldstr: "string1", Fieldint: 42},
				{Fieldstr: "string2", Fieldint: 43},
			}, expected: "FIELDINT\tFIELDSTR\n42\tstring1\n43\tstring2\n"},
			{data: &[]struct1{
				{Fieldstr: "string1", Fieldint: 42},
				{Fieldstr: "string2", Fieldint: 43},
			}, expected: "FIELDINT\tFIELDSTR\n42\tstring1\n43\tstring2\n"},
			{data: []*struct1{
				{Fieldstr: "string1", Fieldint: 42},
				{Fieldstr: "string2", Fieldint: 43},
			}, expected: "FIELDINT\tFIELDSTR\n42\tstring1\n43\tstring2\n"},
			{data: &[]*struct1{
				{Fieldstr: "string1", Fieldint: 42},
				{Fieldstr: "string2", Fieldint: 43},
			}, expected: "FIELDINT\tFIELDSTR\n42\tstring1\n43\tstring2\n"},
			{data: [][]int{}, expected: ""},
			{data: [][]int{{1, 2}, {3}}, expected: "[1 2], [3]"},
			{data: [][]string{{"a1", "a2"}, {"b3"}}, expected: "[a1 a2], [b3]"},
			{data: [][]struct1{
				{{}, {Fieldstr: "a2"}},
				{{Fieldstr: "b3"}},
			}, expected: "[][]printer.struct1{...}"},
			{data: []map[string]struct1{
				{"key1": {}, "key2": {Fieldstr: "a2"}},
				{"key3": {Fieldstr: "b3"}},
			}, expected: "map{key1, key2}\nmap{key3}"},
			{data: []map[string]*struct1{
				{"key1": {}, "key2": {Fieldstr: "a2"}},
				{"key3": {Fieldstr: "b3"}},
			}, expected: "map{key1, key2}\nmap{key3}"},
			{data: []map[string]*struct1{
				{"key2": {}, "key1": {Fieldstr: "a2"}},
				{"key3": {Fieldstr: "b3"}},
			}, expected: "map{key1, key2}\nmap{key3}"},
			{data: []map[int]*struct1{
				{1: {}, 2: {Fieldstr: "a2"}},
				{3: {Fieldstr: "b3"}},
			}, expected: "map{1, 2}\nmap{3}"},
			{data: &[]map[int]*struct1{
				{1: {}, 2: {Fieldstr: "a2"}},
				{3: {Fieldstr: "b3"}},
			}, expected: "map{1, 2}\nmap{3}"},
		} {
			v := reflect.ValueOf(tc.data)
			s := ParseField(v)
			assert.Equal(t, tc.expected, s, fmt.Sprintf("during test case %d: %+v", i, tc.data))
		}
	})
}

func TestStructuredOutput_printTableStruct(t *testing.T) {
	t.Parallel()

	type subStruct struct {
		SubFieldStr string
		SubFieldInt int
		SubSub      *subStruct
	}
	type struct1 struct {
		FieldStr  string
		FieldInt  int
		SubStruct subStruct
	}

	t.Run("complexData", func(t *testing.T) {
		for _, tc := range []struct {
			data     any
			expected string
		}{
			{
				data: struct1{FieldStr: "string", FieldInt: 42, SubStruct: subStruct{SubFieldStr: "substring", SubFieldInt: 43}},
				expected: strings.TrimLeft(`
Field Int:	42
Field Str:	string

Sub Struct:
SUB FIELD INT	SUB FIELD STR	SUB SUB
43	substring	<nil>
`, "\n"),
			},
			{
				data: struct1{FieldStr: "string", FieldInt: 42, SubStruct: subStruct{SubFieldStr: "substring", SubFieldInt: 43, SubSub: &subStruct{SubFieldStr: "subsubstring", SubFieldInt: 44}}},
				expected: strings.TrimLeft(`
Field Int:	42
Field Str:	string

Sub Struct:
SUB FIELD INT	SUB FIELD STR	SUB SUB
43	substring	&{subsubstring 44 <nil>}
`, "\n"),
			},
		} {
			buf := &bytes.Buffer{}
			printTableStruct(buf, tc.data)
			assert.Equal(t, tc.expected, buf.String())
		}
	})
}

func TestGetSortedStructFields(t *testing.T) {
	type s1 struct {
		FieldB     int
		FieldA     int
		unexported int
	}
	t.Run("s1", func(t *testing.T) {
		fields := GetSortedStructFields(reflect.TypeOf(s1{unexported: 1}))
		assert.Equal(t, "FieldA", fields[0].fieldName)
		assert.Equal(t, "FieldB", fields[1].fieldName)
	})

	type s2 struct {
		FieldB int `priority:"99"`
		FieldA int
	}
	t.Run("s2", func(t *testing.T) {
		fields := GetSortedStructFields(reflect.TypeOf(s2{}))
		assert.Equal(t, "FieldB", fields[0].fieldName)
		assert.Equal(t, "FieldA", fields[1].fieldName)
	})

	type s3 struct {
		A int `priority:"-4"`
		B int
		C int `priority:"1"`
	}
	t.Run("s3", func(t *testing.T) {
		fields := GetSortedStructFields(reflect.TypeOf(s3{}))
		assert.Equal(t, []string{"C", "B", "A"}, []string{fields[0].fieldName, fields[1].fieldName, fields[2].fieldName})
	})

	type s4 struct {
		A int `priority:"-4"`
		B int `priority:"8"`
		C int `priority:"8"`
		D int `priority:"invalid"`
		E int `priority:"99999"`
		F int `priority:""`
		G int `priority:"8"`
		H int `priority:"9"`
	}
	t.Run("s4", func(t *testing.T) {
		fields := GetSortedStructFields(reflect.TypeOf(s4{}))
		got := make([]string, len(fields))
		for i, f := range fields {
			got[i] = f.fieldName
		}
		assert.Equal(t, []string{"E", "H", "B", "C", "G", "D", "F", "A"}, got)
	})
}

func TestNormalizeName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in       string
		expected string
	}{
		{in: "field", expected: "Field"},
		{in: "Field", expected: "Field"},
		{in: "fieldName", expected: "Field Name"},
		{in: "FieldName", expected: "Field Name"},
		{in: "FieldName1", expected: "Field Name1"},
		{in: "fieldURL", expected: "Field URL"},
		{in: "fieldURL2", expected: "Field URL2"},
		{in: "fieldWithURL", expected: "Field With URL"},
		{in: "ThisIsSomeField", expected: "This Is Some Field"},
		{in: "SuchAField", expected: "Such A Field"},
		{in: "SuchAFieldLikeThisAgain", expected: "Such A Field Like This Again"},
		{in: "suchAFieldLikeThisAgain", expected: "Such A Field Like This Again"},
	} {
		assert.Equal(t, tc.expected, normalizeName(tc.in))
	}
}
