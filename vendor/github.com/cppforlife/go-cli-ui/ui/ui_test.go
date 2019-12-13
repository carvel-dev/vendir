package ui_test

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui"
	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("UI", func() {
	var (
		logger                   *RecordingLogger
		uiOutBuffer, uiErrBuffer *bytes.Buffer
		uiOut, uiErr             io.Writer
		ui                       UI
	)

	BeforeEach(func() {
		uiOutBuffer = bytes.NewBufferString("")
		uiOut = uiOutBuffer
		uiErrBuffer = bytes.NewBufferString("")
		uiErr = uiErrBuffer
	})

	JustBeforeEach(func() {
		logger = NewRecordingLogger()
		ui = NewWriterUI(uiOut, uiErr, logger)
	})

	Describe("ErrorLinef", func() {
		It("prints to errWriter with a trailing newline", func() {
			ui.ErrorLinef("fake-error-line")
			Expect(uiOutBuffer.String()).To(Equal(""))
			Expect(uiErrBuffer.String()).To(ContainSubstring("fake-error-line\n"))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiErr = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.ErrorLinef("fake-error-line")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logger.ErrOut.String()).To(ContainSubstring("UI.ErrorLinef failed (message='fake-error-line')"))
			})
		})
	})

	Describe("PrintLinef", func() {
		It("prints to outWriter with a trailing newline", func() {
			ui.PrintLinef("fake-line")
			Expect(uiOutBuffer.String()).To(ContainSubstring("fake-line\n"))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.PrintLinef("fake-start")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logger.ErrOut.String()).To(ContainSubstring("UI.PrintLinef failed (message='fake-start')"))
			})
		})
	})

	Describe("BeginLinef", func() {
		It("prints to outWriter", func() {
			ui.BeginLinef("fake-start")
			Expect(uiOutBuffer.String()).To(ContainSubstring("fake-start"))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.BeginLinef("fake-start")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logger.ErrOut.String()).To(ContainSubstring("UI.BeginLinef failed (message='fake-start')"))
			})
		})
	})

	Describe("EndLinef", func() {
		It("prints to outWriter with a trailing newline", func() {
			ui.EndLinef("fake-end")
			Expect(uiOutBuffer.String()).To(ContainSubstring("fake-end\n"))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.EndLinef("fake-start")

				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logger.ErrOut.String()).To(ContainSubstring("UI.EndLinef failed (message='fake-start')"))
			})
		})
	})

	Describe("PrintBlock", func() {
		It("prints to outWriter as is", func() {
			ui.PrintBlock([]byte("block"))
			Expect(uiOutBuffer.String()).To(Equal("block"))
			Expect(uiErrBuffer.String()).To(Equal(""))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.PrintBlock([]byte("block"))
				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logger.ErrOut.String()).To(ContainSubstring("UI.PrintBlock failed (message='block')"))
			})
		})
	})

	Describe("PrintErrorBlock", func() {
		It("prints to outWriter as is", func() {
			ui.PrintErrorBlock("block")
			Expect(uiOutBuffer.String()).To(Equal("block"))
			Expect(uiErrBuffer.String()).To(Equal(""))
		})

		Context("when writing fails", func() {
			BeforeEach(func() {
				reader, writer := io.Pipe()
				uiOut = writer
				reader.Close()
			})

			It("logs an error", func() {
				ui.PrintErrorBlock("block")
				Expect(uiOutBuffer.String()).To(Equal(""))
				Expect(uiErrBuffer.String()).To(Equal(""))
				Expect(logger.ErrOut.String()).To(ContainSubstring("UI.PrintErrorBlock failed (message='block')"))
			})
		})
	})

	Describe("PrintTable", func() {
		It("prints table", func() {
			table := Table{
				Title:   "Title",
				Content: "things",
				Header:  []Header{NewHeader("Header1"), NewHeader("Header2")},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			ui.PrintTable(table)
			Expect("\n" + uiOutBuffer.String()).To(Equal(`
Title

Header1|Header2|
r1c1...|r1c2|
r2c1...|r2c2|

note1
note2

2 things
`))
		})
	})

	Describe("IsInteractive", func() {
		It("returns true", func() {
			Expect(ui.IsInteractive()).To(BeTrue())
		})
	})

	Describe("Flush", func() {
		It("does nothing", func() {
			Expect(func() { ui.Flush() }).ToNot(Panic())
		})
	})
})
