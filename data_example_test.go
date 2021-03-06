package ep_test

import (
	"fmt"
	"github.com/panoplyio/ep"
	"sort"
)

var _ = ep.Types.Register("string", str)
var str = &strType{}

type strType struct{}

func (s *strType) String() string        { return s.Name() }
func (*strType) Name() string            { return "string" }
func (*strType) Data(n int) ep.Data      { return make(strs, n) }
func (*strType) DataEmpty(n int) ep.Data { return make(strs, 0, n) }

type strs []string

func (strs) Type() ep.Type         { return str }
func (vs strs) Len() int           { return len(vs) }
func (vs strs) Less(i, j int) bool { return vs[i] < vs[j] }
func (vs strs) Swap(i, j int)      { vs[i], vs[j] = vs[j], vs[i] }
func (vs strs) LessOther(thisRow int, other ep.Data, otherRow int) bool {
	data := other.(strs)
	return vs[thisRow] < data[otherRow]
}
func (vs strs) Slice(s, e int) ep.Data       { return vs[s:e] }
func (vs strs) Append(other ep.Data) ep.Data { return append(vs, other.(strs)...) }
func (vs strs) Duplicate(t int) ep.Data {
	ans := make(strs, 0, vs.Len()*t)
	for i := 0; i < t; i++ {
		ans = append(ans, vs...)
	}
	return ans
}
func (vs strs) IsNull(i int) bool { return false }
func (vs strs) MarkNull(i int)    {}
func (vs strs) Nulls() []bool     { return make([]bool, vs.Len()) }
func (vs strs) Equal(other ep.Data) bool {
	// for efficiency - avoid reflection and check address of underlying arrays
	return fmt.Sprintf("%p", vs) == fmt.Sprintf("%p", other)
}
func (vs strs) Copy(from ep.Data, fromRow, toRow int) {
	src := from.(strs)
	vs[toRow] = src[fromRow]
}
func (vs strs) Strings() []string { return vs }

func ExampleData() {
	var strs ep.Data = strs([]string{"hello", "world", "foo", "bar"})
	sort.Sort(strs)
	strs = strs.Slice(0, 2)
	fmt.Println(strs.Strings())

	// Output: [bar foo]
}
