package preproc

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
)

var globalSource string = `
func hello1(a string) {
	print(a)
	fmt.Println("zxc")
	println(" hello world!")
}

hello1("zxc")
hello1("vbn")
a := 2

func hello2() bool {
	return True
}

hello2()
`

func TestParse(t *testing.T) {
	type ParseCase struct {
		source   string
		expected error
	}
	cases := []ParseCase{
		// {
		// 	source: globalSource,
		// 	expected: nil,
		// },
		{
			source:   "fmt.Println(\"uzbek\")",
			expected: nil,
		},
	}

	mf := make(map[string]string)
	mv := make(map[string]string)
	mt := make(map[string]string)
	mf["hello3"] = "func(string, string)(string)"
	mv["b"] = "int"
	mt["zxc"] = `type zxc struct {
		a string
		b int
	}`
	types := &KernelTypes{
		vars:  mv,
		funcs: mf,
	}
	for _, testCase := range cases {
		block := NewBlock("228", testCase.source, types)

		err := block.Parse()

		if !errors.Is(err, testCase.expected) {
			t.Fatalf("testparse got error %v, expected %v \n", err, testCase.expected)
		}

		code := block.FormExportFunc()

		fmt.Printf("code: %s", code)
	}
}

func TestMultiple(t *testing.T) {
	type MultipleCase struct {
		sources  []string
		expected error
	}
	cases := []MultipleCase{
		// {
		// 	source: globalSource,
		// 	expected: nil,
		// },
		{
			sources:  []string{"a:=10", "fmt.Println(a)"},
			expected: nil,
		},
	}

	for _, testCase := range cases {
		types := NewKernelTypes()

		for idx, source := range testCase.sources {
			sidx := strconv.Itoa(idx)
			block := NewBlock(sidx, source, types)

			err := block.Parse()

			if !errors.Is(err, testCase.expected) {
				t.Fatalf("testparse got error %v, expected %v \n", err, testCase.expected)
			}

			code := block.FormExportFunc()

			fmt.Printf("code for %s: %s", sidx, code)
		}
	}

}
