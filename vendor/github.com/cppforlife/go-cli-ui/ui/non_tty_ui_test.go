package ui_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui"
	fakeui "github.com/cppforlife/go-cli-ui/ui/fakes"
	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("NonTTYUI", func() {
	var (
		parentUI *fakeui.FakeUI
		ui       UI
	)

	BeforeEach(func() {
		parentUI = &fakeui.FakeUI{}
		ui = NewNonTTYUI(parentUI)
	})

	Describe("ErrorLinef", func() {
		It("includes in Lines", func() {
			ui.ErrorLinef("fake-line1")
			Expect(parentUI.Said).To(BeEmpty())
			Expect(parentUI.Errors).To(Equal([]string{"fake-line1"}))
		})
	})

	Describe("PrintLinef", func() {
		It("does not include in Lines", func() {
			ui.PrintLinef("fake-line1")
			Expect(parentUI.Said).To(BeEmpty())
			Expect(parentUI.Errors).To(BeEmpty())
		})
	})

	Describe("BeginLinef", func() {
		It("does not include in Lines", func() {
			ui.BeginLinef("fake-line1")
			Expect(parentUI.Said).To(BeEmpty())
			Expect(parentUI.Errors).To(BeEmpty())
		})
	})

	Describe("EndLinef", func() {
		It("does not include in Lines", func() {
			ui.EndLinef("fake-line1")
			Expect(parentUI.Said).To(BeEmpty())
			Expect(parentUI.Errors).To(BeEmpty())
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
			ui.PrintBlock([]byte("block"))
			Expect(parentUI.Blocks).To(Equal([]string{"block"}))
		})
	})

	Describe("PrintTable", func() {
		It("delegates to the parent UI with re-configured table", func() {
			ui.PrintTable(Table{
				Title:  "title",
				Header: []Header{NewHeader("header1")},

				Notes:   []string{"note1"},
				Content: "things",

				SortBy: []ColumnSort{{Column: 1}},

				Sections: []Section{
					{
						FirstColumn: ValueString{S: "section1"},
						Rows:        [][]Value{{ValueString{S: "row1"}}},
					},
				},

				Rows: [][]Value{{ValueString{S: "row1"}}},

				FillFirstColumn: false,
				BackgroundStr:   "-",
				BorderStr:       "",
			})

			Expect(parentUI.Table).To(Equal(Table{
				Title: "",
				Header: []Header{
					{Key: "header1", Title: "header1", Hidden: false},
				},
				HeaderFormatFunc: nil,

				Notes:   nil,
				Content: "",

				SortBy: []ColumnSort{{Column: 1}},

				Sections: []Section{
					{
						FirstColumn: ValueString{S: "section1"},
						Rows:        [][]Value{{ValueString{S: "row1"}}},
					},
				},

				Rows: [][]Value{{ValueString{S: "row1"}}},

				FillFirstColumn: true,
				DataOnly:        true,
				BackgroundStr:   "-",
				BorderStr:       "\t",
			}))
		})
	})

	Describe("IsInteractive", func() {
		It("delegates to the parent UI", func() {
			parentUI.Interactive = true
			Expect(ui.IsInteractive()).To(BeTrue())

			parentUI.Interactive = false
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
