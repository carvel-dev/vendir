package table_test

import (
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("Writer", func() {
	var (
		buf                  *bytes.Buffer
		writer               *Writer
		visibleHeaders       []Header
		lastHeaderNotVisible []Header
	)

	BeforeEach(func() {
		buf = bytes.NewBufferString("")
		writer = NewWriter(buf, "empty", ".", "||")
		visibleHeaders = []Header{{Hidden: false}, {Hidden: false}}
		lastHeaderNotVisible = []Header{{Hidden: false}, {Hidden: false}, {Hidden: true}}

	})

	Describe("Write/Flush", func() {
		It("writes single row", func() {
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r0"}, ValueString{S: "c1r0"}})
			writer.Flush()
			Expect(buf.String()).To(Equal("c0r0||c1r0||\n"))
		})

		It("writes multiple rows", func() {
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r0"}, ValueString{S: "c1r0"}})
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r1"}, ValueString{S: "c1r1"}})
			writer.Flush()
			Expect("\n" + buf.String()).To(Equal(`
c0r0||c1r0||
c0r1||c1r1||
`))
		})

		It("writes multiple rows that are not filtered out", func() {
			writer.Write(lastHeaderNotVisible, []Value{ValueString{S: "c0r0"}, ValueString{S: "c1r0"}, ValueString{S: "c2r0"}})
			writer.Write(lastHeaderNotVisible, []Value{ValueString{S: "c0r1"}, ValueString{S: "c1r1"}, ValueString{S: "c2r1"}})
			writer.Flush()
			Expect("\n" + buf.String()).To(Equal(`
c0r0||c1r0||
c0r1||c1r1||
`))
		})

		It("writes every row if not given any headers", func() {
			writer.Write(nil, []Value{ValueString{S: "c0r0"}, ValueString{S: "c1r0"}, ValueString{S: "c1r0"}})
			writer.Write(nil, []Value{ValueString{S: "c0r1"}, ValueString{S: "c1r1"}, ValueString{S: "c2r1"}})
			writer.Flush()
			Expect("\n" + buf.String()).To(Equal(`
c0r0||c1r0||c1r0||
c0r1||c1r1||c2r1||
`))
		})

		It("properly deals with multi-width columns", func() {
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r0-extra"}, ValueString{S: "c1r0"}})
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r1"}, ValueString{S: "c1r1-extra"}})
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r2-extra-extra"}, ValueString{S: "c1r2"}})
			writer.Flush()
			Expect("\n" + buf.String()).To(Equal(`
c0r0-extra......||c1r0||
c0r1............||c1r1-extra||
c0r2-extra-extra||c1r2||
`))
		})

		It("properly deals with multi-width columns and multi-line values", func() {
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r0-extra"}, ValueString{S: "c1r0"}})
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r1\nnext-line"}, ValueString{S: "c1r1-extra"}})
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r2-extra-extra"}, ValueString{S: "c1r2\n\nother\nanother"}})
			writer.Flush()
			Expect("\n" + buf.String()).To(Equal(`
c0r0-extra......||c1r0||
c0r1............||c1r1-extra||
next-line.......||||
c0r2-extra-extra||c1r2||
................||||
................||other||
................||another||
`))
		})

		It("writes empty special value if values are empty", func() {
			writer.Write(visibleHeaders, []Value{ValueString{S: ""}, ValueNone{}})
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r1"}, ValueString{S: "c1r1"}})
			writer.Flush()
			Expect("\n" + buf.String()).To(Equal(`
empty||empty||
c0r1.||c1r1||
`))
		})

		It("uses custom Fprintf for values that support it including multi-line values", func() {
			formattedRegVal := ValueFmt{
				V: ValueString{S: "c0r0"},
				Func: func(pattern string, vals ...interface{}) string {
					return fmt.Sprintf(">%s<", fmt.Sprintf(pattern, vals...))
				},
			}

			formattedMutliVal := ValueFmt{
				V: ValueString{S: "c1r1\n\nother\nanother"},
				Func: func(pattern string, vals ...interface{}) string {
					return fmt.Sprintf(">%s<", fmt.Sprintf(pattern, vals...))
				},
			}

			writer.Write(visibleHeaders, []Value{formattedRegVal, ValueString{S: "c1r0"}})
			writer.Write(visibleHeaders, []Value{ValueString{S: "c0r1"}, formattedMutliVal})
			writer.Flush()

			// Maintains original width for values -- useful for colors since they are not visible
			Expect("\n" + buf.String()).To(Equal(`
>c0r0<||c1r0||
c0r1||>c1r1<||
....||><||
....||>other<||
....||>another<||
`))
		})
	})
})
