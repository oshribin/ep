package ep_test

import (
	"fmt"
	"github.com/panoplyio/ep"
)

func ExampleClone() {
	var d1 ep.Data = strs([]string{"hello", "world"})
	d2 := ep.Clone(d1).(strs)

	d2[0] = "foo"
	d2[1] = "bar"
	fmt.Println(d2) // clone modified
	fmt.Println(d1) // original left intact

	// Output:
	// [foo bar]
	// [hello world]
}

func ExampleCut() {
	var d ep.Data = strs([]string{"hello", "world", "foo", "bar"})
	data := ep.Cut(d, 1, 3)
	fmt.Println(data.Strings())

	// Output:
	// [[hello] [world foo] [bar]]
}
