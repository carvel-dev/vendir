package test_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cppforlife/go-cli-ui/ui"
	. "github.com/cppforlife/go-cli-ui/ui/test"
)

var _ = Describe("JSONUIFromBytes", func() {
	const (
		example = `
{
    "Tables": [
        {
            "Content": "services",
            "Header": {
                "created_at": "Created At",
                "domain": "Domain",
                "internal_domain": "Internal Domain",
                "name": "Name"
            },
            "Rows": [
                {
                    "created_at": "2018-07-31T12:27:45-07:00",
                    "domain": "foo3.default.example.com",
                    "internal_domain": "foo3.default.svc.cluster.local",
                    "name": "foo3"
                },
                {
                    "created_at": "2018-07-26T16:47:39-07:00",
                    "domain": "helloworld-go.default.example.com",
                    "internal_domain": "helloworld-go.default.svc.cluster.local",
                    "name": "helloworld-go"
                }
            ],
            "Notes": null
        }
    ],
    "Blocks": null,
    "Lines": null
}
`
	)

	It("properly parses JSON UI output", func() {
		resp := JSONUIFromBytes(nil, []byte(example))
		Expect(resp).To(Equal(ui.JSONUIResp{
			Tables: []ui.JSONUITableResp{
				{
					Content: "services",
					Header: map[string]string{
						"domain":          "Domain",
						"internal_domain": "Internal Domain",
						"name":            "Name",
						"created_at":      "Created At",
					},
					Rows: []map[string]string{
						{
							"internal_domain": "foo3.default.svc.cluster.local",
							"name":            "foo3",
							"created_at":      "2018-07-31T12:27:45-07:00",
							"domain":          "foo3.default.example.com",
						},
						{
							"name":            "helloworld-go",
							"created_at":      "2018-07-26T16:47:39-07:00",
							"domain":          "helloworld-go.default.example.com",
							"internal_domain": "helloworld-go.default.svc.cluster.local",
						},
					},
					Notes: nil,
				},
			},
			Blocks: nil,
			Lines:  nil,
		}))
	})
})
