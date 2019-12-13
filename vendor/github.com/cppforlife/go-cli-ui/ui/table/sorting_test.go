package table_test

import (
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("Sorting", func() {
	It("sorts by single column in asc order", func() {
		sortBy := []ColumnSort{{Column: 0, Asc: true}}
		rows := [][]Value{
			{ValueString{S: "b"}, ValueString{S: "x"}},
			{ValueString{S: "a"}, ValueString{S: "y"}},
		}

		sort.Sort(Sorting{SortBy: sortBy, Rows: rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{S: "a"}, ValueString{S: "y"}},
			{ValueString{S: "b"}, ValueString{S: "x"}},
		}))
	})

	It("sorts by single column in desc order", func() {
		sortBy := []ColumnSort{{Column: 0, Asc: false}}
		rows := [][]Value{
			{ValueString{S: "a"}, ValueString{S: "y"}},
			{ValueString{S: "b"}, ValueString{S: "x"}},
		}

		sort.Sort(Sorting{SortBy: sortBy, Rows: rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{S: "b"}, ValueString{S: "x"}},
			{ValueString{S: "a"}, ValueString{S: "y"}},
		}))
	})

	It("sorts by multiple columns in asc order", func() {
		sortBy := []ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: true},
		}

		rows := [][]Value{
			{ValueString{S: "b"}, ValueString{S: "x"}, ValueString{S: "2"}},
			{ValueString{S: "a"}, ValueString{S: "y"}, ValueString{S: "1"}},
			{ValueString{S: "b"}, ValueString{S: "z"}, ValueString{S: "2"}},
			{ValueString{S: "c"}, ValueString{S: "t"}, ValueString{S: "0"}},
		}

		sort.Sort(Sorting{SortBy: sortBy, Rows: rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{S: "a"}, ValueString{S: "y"}, ValueString{S: "1"}},
			{ValueString{S: "b"}, ValueString{S: "x"}, ValueString{S: "2"}},
			{ValueString{S: "b"}, ValueString{S: "z"}, ValueString{S: "2"}},
			{ValueString{S: "c"}, ValueString{S: "t"}, ValueString{S: "0"}},
		}))
	})

	It("sorts by multiple columns in asc and desc order", func() {
		sortBy := []ColumnSort{
			{Column: 0, Asc: false},
			{Column: 1, Asc: true},
		}

		rows := [][]Value{
			{ValueString{S: "b"}, ValueString{S: "z"}, ValueString{S: "2"}},
			{ValueString{S: "a"}, ValueString{S: "x"}, ValueString{S: "1"}},
			{ValueString{S: "b"}, ValueString{S: "y"}, ValueString{S: "2"}},
			{ValueString{S: "c"}, ValueString{S: "t"}, ValueString{S: "0"}},
		}

		sort.Sort(Sorting{SortBy: sortBy, Rows: rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{S: "c"}, ValueString{S: "t"}, ValueString{S: "0"}},
			{ValueString{S: "b"}, ValueString{S: "y"}, ValueString{S: "2"}},
			{ValueString{S: "b"}, ValueString{S: "z"}, ValueString{S: "2"}},
			{ValueString{S: "a"}, ValueString{S: "x"}, ValueString{S: "1"}},
		}))
	})

	It("sorts real values (e.g. suffix does not count)", func() {
		sortBy := []ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: true},
		}

		rows := [][]Value{
			{ValueSuffix{V: ValueString{S: "a"}, Suffix: "b"}, ValueString{S: "x"}},
			{ValueSuffix{V: ValueString{S: "a"}, Suffix: "a"}, ValueString{S: "y"}},
		}

		sort.Sort(Sorting{SortBy: sortBy, Rows: rows})

		Expect(rows).To(Equal([][]Value{
			{ValueSuffix{V: ValueString{S: "a"}, Suffix: "b"}, ValueString{S: "x"}},
			{ValueSuffix{V: ValueString{S: "a"}, Suffix: "a"}, ValueString{S: "y"}},
		}))
	})
})
