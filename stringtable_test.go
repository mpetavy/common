package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStringTable(t *testing.T) {
	st := NewStringTable()

	st.AddRow()
	st.AddCol("aaaa")
	st.AddCol("aaa")
	st.AddCol("aa")
	st.AddCol("a")
	st.AddCols("b", "bbb", "bb", "bbbb")
	st.AddCols("X", "X", "X", "X")

	require.Equal(t, `aaaa | aaa | aa | a   
-----+-----+----+-----
b    | bbb | bb | bbbb
X    | X   | X  | X   
`, st.Table())

	require.Equal(t, `| aaaa | aaa | aa | a    |
| ---- | --- | -- | ---- |
| b    | bbb | bb | bbbb |
| X    | X   | X  | X    |
`, st.Markdown())

	require.Equal(t, `<table>
	<thead>
	<tr>
		<td>aaaa</td>
		<td>aaa</td>
		<td>aa</td>
		<td>a</td>
	</tr>
	</thead>
	<tr>
		<td>b</td>
		<td>bbb</td>
		<td>bb</td>
		<td>bbbb</td>
	</tr>
	<tr>
		<td>X</td>
		<td>X</td>
		<td>X</td>
		<td>X</td>
	</tr>
</table>
`, st.Html())

	require.Equal(t, `[
    {
        "aaaa": "b",
        "aaa": "bbb",
        "aa": "bb",
        "a": "bbbb"
    },
    {
        "aaaa": "X",
        "aaa": "X",
        "aa": "X",
        "a": "X"
    }
]
`, st.JSON("    "))

	require.Equal(t, `"aaaa","aaa","aa","a"
"b","bbb","bb","bbbb"
"X","X","X","X"
`, st.CSV())
}
