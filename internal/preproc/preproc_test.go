package preproc

import (
	"errors"
	"fmt"
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
		source string 
		expected error
	}
	cases := []ParseCase{
		{
			source: globalSource,
			expected: nil,
		},
	}

	for _, testCase := range(cases) {
		block := NewBlock("228", testCase.source)

		err := block.Parse()

		if !errors.Is(err, testCase.expected) {
			t.Fatalf("testparse got error %v, expected %v \n", err, testCase.expected)
		}

		code := block.FormExportFunc()

		fmt.Printf("code: %s", code)
	}
}