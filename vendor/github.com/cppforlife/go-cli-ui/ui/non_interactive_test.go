package ui_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui"
	fakeui "github.com/cppforlife/go-cli-ui/ui/fakes"
	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("NonInteractiveUI", func() {
	var (
		parentUI *fakeui.FakeUI
		ui       UI
	)

	BeforeEach(func() {
		parentUI = &fakeui.FakeUI{}
		ui = NewNonInteractiveUI(parentUI)
	})

	Describe("ErrorLinef", func() {
		It("delegates to the parent UI", func() {
			ui.ErrorLinef("fake-error-line")
			Expect(parentUI.Errors).To(Equal([]string{"fake-error-line"}))
		})
	})

	Describe("PrintLinef", func() {
		It("delegates to the parent UI", func() {
			ui.PrintLinef("fake-line")
			Expect(parentUI.Said).To(Equal([]string{"fake-line"}))
		})
	})

	Describe("BeginLinef", func() {
		It("delegates to the parent UI", func() {
			ui.BeginLinef("fake-start")
			Expect(parentUI.Said).To(Equal([]string{"fake-start"}))
		})
	})

	Describe("EndLinef", func() {
		It("delegates to the parent UI", func() {
			ui.EndLinef("fake-end")
			Expect(parentUI.Said).To(Equal([]string{"fake-end"}))
		})
	})

	Describe("PrintBlock", func() {
		It("delegates to the parent UI", func() {
			ui.PrintBlock([]byte("block"))
			Expect(parentUI.Blocks).To(Equal([]string{"block"}))
		})
	})

	Describe("PrintErrorBlock", func() {
		It("delegates to the parent UI", func() {
			ui.PrintErrorBlock("block")
			Expect(parentUI.Blocks).To(Equal([]string{"block"}))
		})
	})

	Describe("PrintTable", func() {
		It("delegates to the parent UI", func() {
			table := Table{
				Content: "things",
				Header:  []Header{NewHeader("header1")},
			}

			ui.PrintTable(table)

			Expect(parentUI.Table).To(Equal(table))
		})
	})

	Describe("AskForText", func() {
		It("panics", func() {
			Expect(func() { ui.AskForText("") }).To(Panic())
		})
	})

	Describe("AskForPassword", func() {
		It("panics", func() {
			Expect(func() { ui.AskForPassword("") }).To(Panic())
		})
	})

	Describe("AskForChoice", func() {
		It("panics", func() {
			Expect(func() { ui.AskForChoice("", nil) }).To(Panic())
		})
	})

	Describe("AskForConfirmation", func() {
		It("responds affirmatively with no error", func() {
			Expect(ui.AskForConfirmation()).To(BeNil())
		})
	})

	Describe("IsInteractive", func() {
		It("returns false", func() {
			Expect(ui.IsInteractive()).To(BeFalse())
		})
	})

	Describe("Flush", func() {
		It("delegates to the parent UI", func() {
			ui.Flush()
			Expect(parentUI.Flushed).To(BeTrue())
		})
	})
})
