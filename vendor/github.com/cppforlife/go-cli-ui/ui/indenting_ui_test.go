package ui_test

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui"
	fakeui "github.com/cppforlife/go-cli-ui/ui/fakes"
	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("IndentingUI", func() {
	var (
		uiOut, uiErr *bytes.Buffer
		parentFakeUI *fakeui.FakeUI
		parentUI     UI
		ui           UI
	)

	BeforeEach(func() {
		uiOut = bytes.NewBufferString("")
		uiErr = bytes.NewBufferString("")

		logger := NewRecordingLogger()
		parentUI = NewWriterUI(uiOut, uiErr, logger)
		parentFakeUI = &fakeui.FakeUI{}
	})

	JustBeforeEach(func() {
		ui = NewIndentingUI(parentUI)
	})

	Describe("ErrorLinef", func() {
		It("delegates to the parent UI with an indent", func() {
			ui.ErrorLinef("fake-error-line")
			Expect(uiErr.String()).To(ContainSubstring("  fake-error-line\n"))
			Expect(uiOut.String()).To(BeEmpty())
		})
	})

	Describe("PrintLinef", func() {
		It("delegates to the parent UI with an indent", func() {
			ui.PrintLinef("fake-line")
			Expect(uiOut.String()).To(ContainSubstring("  fake-line\n"))
			Expect(uiErr.String()).To(BeEmpty())
		})
	})

	Describe("BeginLinef", func() {
		It("delegates to the parent UI with an indent", func() {
			ui.BeginLinef("fake-start")
			Expect(uiOut.String()).To(ContainSubstring("  fake-start"))
			Expect(uiErr.String()).To(BeEmpty())
		})
	})

	Describe("EndLinef", func() {
		It("delegates to the parent UI", func() {
			ui.EndLinef("fake-end")
			Expect(uiOut.String()).To(ContainSubstring("fake-end\n"))
			Expect(uiErr.String()).To(BeEmpty())
		})
	})

	Describe("PrintBlock", func() {
		BeforeEach(func() {
			parentUI = parentFakeUI
		})

		It("delegates to the parent UI", func() {
			ui.PrintBlock([]byte("block"))
			Expect(parentFakeUI.Blocks).To(Equal([]string{"block"}))
		})
	})

	Describe("PrintErrorBlock", func() {
		BeforeEach(func() {
			parentUI = parentFakeUI
		})

		It("delegates to the parent UI", func() {
			ui.PrintBlock([]byte("block"))
			Expect(parentFakeUI.Blocks).To(Equal([]string{"block"}))
		})
	})

	Describe("PrintTable", func() {
		BeforeEach(func() {
			parentUI = parentFakeUI
		})

		It("delegates to the parent UI", func() {
			table := Table{
				Content: "things",
				Header:  []Header{NewHeader("header1")},
			}

			ui.PrintTable(table)

			Expect(parentFakeUI.Table).To(Equal(table))
		})
	})

	Describe("IsInteractive", func() {
		BeforeEach(func() {
			parentUI = parentFakeUI
		})

		It("delegates to the parent UI", func() {
			parentFakeUI.Interactive = true
			Expect(ui.IsInteractive()).To(BeTrue())

			parentFakeUI.Interactive = false
			Expect(ui.IsInteractive()).To(BeFalse())
		})
	})

	Describe("Flush", func() {
		BeforeEach(func() {
			parentUI = parentFakeUI
		})

		It("delegates to the parent UI", func() {
			ui.Flush()
			Expect(parentFakeUI.Flushed).To(BeTrue())
		})
	})
})
