package ui_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cppforlife/go-cli-ui/ui"
	fakeui "github.com/cppforlife/go-cli-ui/ui/fakes"
	. "github.com/cppforlife/go-cli-ui/ui/table"
)

var _ = Describe("JSONUI", func() {
	var (
		parentUI *fakeui.FakeUI
		ui       UI
	)

	BeforeEach(func() {
		parentUI = &fakeui.FakeUI{}
		logger := NewRecordingLogger()
		ui = NewJSONUI(parentUI, logger)
	})

	type tableResp struct {
		Content string
		Header  map[string]string
		Rows    []map[string]string
		Notes   []string
	}

	type uiResp struct {
		Tables []tableResp
		Blocks []string
		Lines  []string
	}

	finalOutput := func() uiResp {
		ui.Flush()

		var val uiResp

		err := json.Unmarshal([]byte(parentUI.Blocks[0]), &val)
		if err != nil {
			panic("Unmarshaling")
		}

		return val
	}

	Describe("ErrorLinef", func() {
		It("includes in Lines", func() {
			ui.ErrorLinef("fake-line1")
			ui.ErrorLinef("fake-line2")
			Expect(finalOutput()).To(Equal(uiResp{
				Lines: []string{"fake-line1", "fake-line2"},
			}))
		})
	})

	Describe("PrintLinef", func() {
		It("includes in Lines", func() {
			ui.PrintLinef("fake-line1")
			ui.PrintLinef("fake-line2")
			Expect(finalOutput()).To(Equal(uiResp{
				Lines: []string{"fake-line1", "fake-line2"},
			}))
		})
	})

	Describe("BeginLinef", func() {
		It("includes in Lines", func() {
			ui.BeginLinef("fake-line1")
			ui.BeginLinef("fake-line2")
			Expect(finalOutput()).To(Equal(uiResp{
				Lines: []string{"fake-line1", "fake-line2"},
			}))
		})
	})

	Describe("EndLinef", func() {
		It("includes in Lines", func() {
			ui.EndLinef("fake-line1")
			ui.EndLinef("fake-line2")
			Expect(finalOutput()).To(Equal(uiResp{
				Lines: []string{"fake-line1", "fake-line2"},
			}))
		})
	})

	Describe("PrintBlock", func() {
		It("includes in Blocks", func() {
			ui.PrintBlock([]byte("fake-block1"))
			ui.PrintBlock([]byte("fake-block2"))
			Expect(finalOutput()).To(Equal(uiResp{
				Blocks: []string{"fake-block1", "fake-block2"},
			}))
		})
	})

	Describe("PrintErrorBlock", func() {
		It("includes in Blocks", func() {
			ui.PrintErrorBlock("fake-block1")
			ui.PrintErrorBlock("fake-block2")
			Expect(finalOutput()).To(Equal(uiResp{
				Blocks: []string{"fake-block1", "fake-block2"},
			}))
		})
	})

	Describe("PrintTable", func() {
		It("includes table response in Tables", func() {
			table := Table{
				Content: "things",
				Header:  []Header{NewHeader("Header & ( foo )  1 "), NewHeader("Header-2 header 3")},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				Notes: []string{"note1", "note2"},
			}

			table2 := Table{
				Content: "things2",
			}

			ui.PrintTable(table)
			ui.PrintTable(table2)

			Expect(finalOutput()).To(Equal(uiResp{
				Tables: []tableResp{
					{
						Content: "things",
						Header:  map[string]string{"header_foo_1": "Header & ( foo )  1 ", "header_2_header_3": "Header-2 header 3"},
						Rows: []map[string]string{
							{"header_foo_1": "r1c1", "header_2_header_3": "r1c2"},
							{"header_foo_1": "r2c1", "header_2_header_3": "r2c2"},
						},
						Notes: []string{"note1", "note2"},
					},
					{
						Content: "things2",
						Header:  map[string]string{},
						Rows:    []map[string]string{},
					},
				},
			}))
		})

		It("generates header keys for tables with row content and no header content", func() {
			table := Table{
				Content: "things",
				Header:  []Header{},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}},
				},

				Notes: []string{"note1", "note2"},
			}

			ui.PrintTable(table)

			Expect(finalOutput()).To(Equal(uiResp{
				Tables: []tableResp{
					{
						Content: "things",
						Header:  map[string]string{"col_0": "", "col_1": ""},
						Rows: []map[string]string{
							{"col_0": "r1c1", "col_1": "r1c2"},
							{"col_0": "r2c1", "col_1": "r2c2"},
						},
						Notes: []string{"note1", "note2"},
					},
				},
			}))
		})

		It("includes Headers in Tables", func() {
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

			table2 := Table{
				Content: "things2",
			}

			ui.PrintTable(table)
			ui.PrintTable(table2)

			Expect(finalOutput()).To(Equal(uiResp{
				Tables: []tableResp{
					{
						Content: "things",
						Header:  map[string]string{"header1": "Header1", "header2": "Header2"},
						Rows: []map[string]string{{"header1": "r1c1", "header2": "r1c2"},
							{"header1": "r2c1", "header2": "r2c2"}},
						Notes: []string{"note1", "note2"},
					},
					{
						Content: "things2",
						Header:  map[string]string{},
						Rows:    []map[string]string{},
					},
				},
			}))
		})

		It("convert non-alphanumeric to _", func() {
			table := Table{
				Content: "things",
				Header: []Header{
					NewHeader("#"),
					NewHeader("foo"),
					NewHeader("$"),
				},

				Rows: [][]Value{
					{ValueString{S: "r1c1"}, ValueString{S: "r1c2"}, ValueString{S: "r1c3"}},
					{ValueString{S: "r2c1"}, ValueString{S: "r2c2"}, ValueString{S: "r2c3"}},
				},

				Notes: []string{},
			}

			ui.PrintTable(table)

			tableOutput := finalOutput()
			Expect(tableOutput.Tables).To(HaveLen(1))
			Expect(tableOutput.Tables[0].Content).To(Equal("things"))
			Expect(tableOutput.Tables[0].Header).To(HaveKeyWithValue("0", "#"))
			Expect(tableOutput.Tables[0].Header).To(HaveKeyWithValue("foo", "foo"))
			Expect(tableOutput.Tables[0].Header).To(HaveKeyWithValue("2", "$"))

			Expect(tableOutput.Tables[0].Rows).To(HaveLen(2))
			Expect(tableOutput.Tables[0].Rows[0]).To(HaveKeyWithValue("0", "r1c1"))
			Expect(tableOutput.Tables[0].Rows[0]).To(HaveKeyWithValue("foo", "r1c2"))
			Expect(tableOutput.Tables[0].Rows[0]).To(HaveKeyWithValue("2", "r1c3"))

			Expect(tableOutput.Tables[0].Rows[1]).To(HaveKeyWithValue("0", "r2c1"))
			Expect(tableOutput.Tables[0].Rows[1]).To(HaveKeyWithValue("foo", "r2c2"))
			Expect(tableOutput.Tables[0].Rows[1]).To(HaveKeyWithValue("2", "r2c3"))
		})

		It("includes in Tables when table has sections and fills in first column", func() {
			table := Table{
				Content: "things",
				Header:  []Header{NewHeader("Header1"), NewHeader("Header2")},

				Sections: []Section{
					{
						FirstColumn: ValueString{S: "first-col"},
						Rows: [][]Value{
							{ValueString{S: ""}, ValueString{S: "r1c2"}},
							{ValueString{S: ""}, ValueString{S: "r2c2"}},
						},
					},
				},

				Notes: []string{"note1", "note2"},
			}

			ui.PrintTable(table)

			Expect(finalOutput()).To(Equal(uiResp{
				Tables: []tableResp{
					{
						Content: "things",
						Header:  map[string]string{"header1": "Header1", "header2": "Header2"},
						Rows: []map[string]string{{"header1": "first-col", "header2": "r1c2"},
							{"header1": "first-col", "header2": "r2c2"}},
						Notes: []string{"note1", "note2"},
					},
				},
			}))
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
		It("panics", func() {
			Expect(func() { ui.AskForConfirmation() }).To(Panic())
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
		It("does not output anything when nothing was recorded", func() {
			ui.Flush()
			Expect(parentUI.Said).To(BeEmpty())
		})

		It("outputs everything when something was recorded", func() {
			ui.PrintLinef("fake-line1")
			ui.Flush()
			Expect(parentUI.Blocks[0]).To(Equal(`{
    "Tables": null,
    "Blocks": null,
    "Lines": [
        "fake-line1"
    ]
}`))
		})
	})
})
