package table_test

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("Table", func() {
	var (
		buf *bytes.Buffer
	)

	BeforeEach(func() {
		buf = bytes.NewBufferString("")
	})

	Describe("Print", func() {
		It("prints a table in default formatting (borders, empties, etc.)", func() {
			table := Table{
				Content: "things",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				Notes: []string{"note1", "note2"},
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(strings.Replace(`
Header1  Header2  +
r1c1     r1c2  +
r2c1     r2c2  +

note1
note2

2 things
`, "+", "", -1)))
		})

		It("prints a table with header if Header is specified", func() {
			table := Table{
				Content: "things",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
r1c1...|r1c2|
r2c1...|r2c2|

note1
note2

2 things
`))
		})

		It("prints a table without number of records if content is not specified", func() {
			table := Table{
				Content: "",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
r1c1...|r1c2|
r2c1...|r2c2|

note1
note2
`))
		})

		It("prints a table sorted based on SortBy", func() {
			table := Table{
				SortBy: []ColumnSort{{Column: 1}, {Column: 0, Asc: true}},

				Rows: [][]Value{
					{ValueString{S: "a"}, ValueInt{I: -1}},
					{ValueString{S: "b"}, ValueInt{I: 0}},
					{ValueString{S: "d"}, ValueInt{I: 20}},
					{ValueString{S: "c"}, ValueInt{I: 20}},
					{ValueString{S: "d"}, ValueInt{I: 100}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
d|100|
c|20|
d|20|
b|0|
a|-1|
`))
		})

		It("prints a table without a header if Header is not specified", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
r2c1|r2c2|
`))
		})

		It("prints a table with a title and a header", func() {
			table := Table{
				Title:   "Title",
				Content: "things",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Title

Header1|Header2|
r1c1...|r1c2|
r2c1...|r2c2|

note1
note2

2 things
`))
		})

		Context("when sections are provided", func() {
			It("prints a table *without* sections for now", func() {
				table := Table{
					Content: "things",
					Sections: []Section{
						{
							Rows: [][]Value{
								{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
							},
						},
					},
					BackgroundStr: ".",
					BorderStr:     "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
r2c1|r2c2|
`))
			})

			It("prints a table with first column set", func() {
				table := Table{
					Content: "things",
					Sections: []Section{
						{
							FirstColumn: ValueString{S: "r1c1"},

							Rows: [][]Value{
								{ValueString{S: ""}, ValueString{S: "r1c2"}},
								{ValueString{S: ""}, ValueString{S: "r2c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{S: "r3c1"}, ValueString{S: "r3c2"}},
							},
						},
					},
					BackgroundStr: ".",
					BorderStr:     "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
~...|r2c2|
r3c1|r3c2|
`))
			})

			It("prints a table with first column filled for all rows when option is set", func() {
				table := Table{
					Content: "things",
					Sections: []Section{
						{
							FirstColumn: ValueString{S: "r1c1"},
							Rows: [][]Value{
								{ValueString{S: ""}, ValueString{S: "r1c2"}},
								{ValueString{S: ""}, ValueString{S: "r2c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{S: "r3c1"}, ValueString{S: "r3c2"}},
							},
						},
						{
							FirstColumn: ValueString{S: "r4c1"},
							Rows: [][]Value{
								{ValueString{S: ""}, ValueString{S: "r4c2"}},
								{ValueString{S: ""}, ValueString{S: "r5c2"}},
								{ValueString{S: ""}, ValueString{S: "r6c2"}},
							},
						},
					},
					FillFirstColumn: true,
					BackgroundStr:   ".",
					BorderStr:       "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
r1c1|r2c2|
r3c1|r3c2|
r4c1|r4c2|
r4c1|r5c2|
r4c1|r6c2|
`))
			})

			It("prints a footer including the counts for rows in sections", func() {
				table := Table{
					Content: "things",
					Header: []Header{
						NewHeader("Header1"),
						NewHeader("Header2"),
					},
					Sections: []Section{
						{
							FirstColumn: ValueString{S: "s1c1"},
							Rows: [][]Value{
								{ValueString{S: ""}, ValueString{S: "s1r1c2"}},
								{ValueString{S: ""}, ValueString{S: "s1r2c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{S: "r3c1"}, ValueString{S: "r3c2"}},
							},
						},
					},
					Rows: [][]Value{
						{ValueString{S: "r4c1"}, ValueString{S: "r4c2"}},
					},
					FillFirstColumn: true,
					BackgroundStr:   ".",
					BorderStr:       "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
s1c1...|s1r1c2|
s1c1...|s1r2c2|
r3c1...|r3c2|
r4c1...|r4c2|

4 things
`))
			})
		})

		It("prints values in table that span multiple lines", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2.1\nr1c2.2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2.1|
....|r1c2.2|
r2c1|r2c2|
`))
		})

		It("removes duplicate values in the first column", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{S: "dup"}, ValueString{S: "dup"}},
					{ValueString{S: "dup"}, ValueString{S: "dup"}},
					{ValueString{S: "dup2"}, ValueString{S: "dup"}},
					{ValueString{S: "dup2"}, ValueString{S: "dup"}},
					{ValueString{S: "other"}, ValueString{S: "dup"}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup..|dup|
~....|dup|
dup2.|dup|
~....|dup|
other|dup|
`))
		})

		It("does not removes duplicate values in the first column if FillFirstColumn is true", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{S: "dup"}, ValueString{S: "dup"}},
					{ValueString{S: "dup"}, ValueString{S: "dup"}},
					{ValueString{S: "dup2"}, ValueString{S: "dup"}},
					{ValueString{S: "dup2"}, ValueString{S: "dup"}},
					{ValueString{S: "other"}, ValueString{S: "dup"}},
				},

				FillFirstColumn: true,
				BackgroundStr:   ".",
				BorderStr:       "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup..|dup|
dup..|dup|
dup2.|dup|
dup2.|dup|
other|dup|
`))
		})

		It("removes duplicate values in the first column even with sections", func() {
			table := Table{
				Content: "things",

				Sections: []Section{
					{
						FirstColumn: ValueString{S: "dup"},
						Rows: [][]Value{
							{ValueNone{}, ValueString{S: "dup"}},
							{ValueNone{}, ValueString{S: "dup"}},
						},
					},
					{
						FirstColumn: ValueString{S: "dup2"},
						Rows: [][]Value{
							{ValueNone{}, ValueString{S: "dup"}},
							{ValueNone{}, ValueString{S: "dup"}},
						},
					},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup.|dup|
~...|dup|
dup2|dup|
~...|dup|
`))
		})

		It("removes duplicate values in the first column after sorting", func() {
			table := Table{
				Content: "things",

				SortBy: []ColumnSort{{Column: 1, Asc: true}},

				Rows: [][]Value{
					{ValueString{S: "dup"}, ValueInt{I: 1}},
					{ValueString{S: "dup2"}, ValueInt{I: 3}},
					{ValueString{S: "dup"}, ValueInt{I: 2}},
					{ValueString{S: "dup2"}, ValueInt{I: 4}},
					{ValueString{S: "other"}, ValueInt{I: 5}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup..|1|
~....|2|
dup2.|3|
~....|4|
other|5|
`))
		})

		It("prints empty values as dashes", func() {
			table := Table{
				Rows: [][]Value{
					{ValueString{S: ""}, ValueNone{}},
					{ValueString{S: ""}, ValueNone{}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
-|-|
~|-|
`))
		})

		It("prints empty tables without rows and section", func() {
			table := Table{
				Content: "content",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|

0 content
`))
		})

		Context("table has Transpose:true", func() {
			It("prints as transposed table", func() {
				table := Table{
					Content: "errands",
					Header: []Header{
						NewHeader("Header1"),
						NewHeader("OtherHeader2"),
						NewHeader("Header3"),
					},
					Rows: [][]Value{
						{ValueString{S: "r1c1"}, ValueString{S: "longr1c2"}, ValueString{S: "r1c3"}},
						{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}, ValueString{S: "r2c3"}},
					},
					BackgroundStr: ".",
					BorderStr:     "|",
					Transpose:     true,
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1.....|r1c1|
OtherHeader2|longr1c2|
Header3.....|r1c3|

Header1.....|r2c1|
OtherHeader2|r2c2|
Header3.....|r2c3|

2 errands
`))
			})

			It("prints a filtered transposed table", func() {
				nonVisibleHeader := NewHeader("Header3")
				nonVisibleHeader.Hidden = true

				table := Table{
					Content: "errands",

					Header: []Header{
						NewHeader("Header1"),
						NewHeader("Header2"),
						nonVisibleHeader,
					},
					Rows: [][]Value{
						{ValueString{S: "v1"}, ValueString{S: "v2"}, ValueString{S: "v3"}},
					},
					BorderStr: "|",
					Transpose: true,
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1|v1|
Header2|v2|

1 errands
`))
			})

			Context("when table also has a SortBy value set", func() {
				It("prints as transposed table with sections sorted by the SortBy", func() {
					table := Table{
						Content: "errands",
						Header: []Header{
							NewHeader("Header1"),
							NewHeader("OtherHeader2"),
							NewHeader("Header3"),
						},
						Rows: [][]Value{
							{ValueString{S: "r1c1"}, ValueString{S: "longr1c2"}, ValueString{S: "r1c3"}},
							{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}, ValueString{S: "r2c3"}},
						},
						SortBy: []ColumnSort{
							{Column: 0, Asc: true},
						},
						BackgroundStr: ".",
						BorderStr:     "|",
						Transpose:     true,
					}
					table.Print(buf)
					Expect("\n" + buf.String()).To(Equal(`
Header1.....|r1c1|
OtherHeader2|longr1c2|
Header3.....|r1c3|

Header1.....|r2c1|
OtherHeader2|r2c2|
Header3.....|r2c3|

2 errands
`))
				})
			})
		})

		Context("when column filtering is used", func() {
			It("prints all non-filtered out columns", func() {
				nonVisibleHeader := NewHeader("Header3")
				nonVisibleHeader.Hidden = true

				table := Table{
					Content: "content",

					Header: []Header{
						NewHeader("Header1"),
						NewHeader("Header2"),
						nonVisibleHeader,
					},
					Rows: [][]Value{
						{ValueString{S: "v1"}, ValueString{S: "v2"}, ValueString{S: "v3"}},
					},
					BorderStr: "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
v1     |v2|

1 content
`))
			})
		})
	})

	Describe("AddColumn", func() {
		It("returns an updated table with the new column", func() {
			table := Table{
				Content: "content",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},
				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},
				BackgroundStr: ".",
				BorderStr:     "|",
			}

			newTable := table.AddColumn("Header3", []Value{ValueString{S: "r1c3"}, ValueString{S: "r2c3"}})
			Expect(newTable).To(Equal(Table{
				Content: "content",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
					NewHeader("Header3"),
				},
				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}, ValueString{S: "r1c3"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}, ValueString{S: "r2c3"}},
				},
				BackgroundStr: ".",
				BorderStr:     "|",
			}))
		})
	})
})
