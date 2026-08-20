package domain_test

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestDomain(t *testing.T) {
	t.Parallel()

	d := "exoscale_domain.test_domain"
	dDS := "data.exoscale_domain.test_domain"
	mx := "exoscale_domain_record.test_mx"
	txt := "exoscale_domain_record.test_txt"
	byName := "data.exoscale_domain_record.by_name"
	byID := "data.exoscale_domain_record.by_id"
	byRegex := "data.exoscale_domain_record.by_regex"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}
	domainName := testutils.ResourceName(testdataSpec.ID) + ".net"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create domain and records, exercise the data sources.
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/001.domain_create.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(d, "name", domainName),
					resource.TestCheckResourceAttrPair(d, "name", dDS, "name"),
					resource.TestCheckResourceAttrPair(d, "id", dDS, "id"),

					resource.TestCheckResourceAttr(mx, "name", "mail1"),
					resource.TestCheckResourceAttr(mx, "record_type", "MX"),
					resource.TestCheckResourceAttr(mx, "content", "mta1."+domainName),
					resource.TestCheckResourceAttr(mx, "content_normalized", "mta1."+domainName),
					resource.TestCheckResourceAttr(mx, "prio", "10"),
					resource.TestCheckResourceAttr(mx, "ttl", "10"),
					resource.TestCheckResourceAttr(mx, "hostname", "mail1."+domainName),

					resource.TestCheckResourceAttr(txt, "content", "test value for TXT record"),
					resource.TestCheckResourceAttr(txt, "content_normalized", `"test value for TXT record"`),

					resource.TestCheckResourceAttr(byName, "records.0.name", "mail1"),
					resource.TestCheckResourceAttr(byID, "records.0.name", "mail1"),
					resource.TestCheckResourceAttr(byRegex, "records.0.content", "mta1."+domainName),
				),
			},

			// 2 Update the MX record, and let the TXT record content self-heal
			// once its remote (API-normalized) content differs from state.
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/002.domain_record_update.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(mx, "name", "mail2"),
					resource.TestCheckResourceAttr(mx, "content", "mta2."+domainName),
					resource.TestCheckResourceAttr(mx, "prio", "20"),
					resource.TestCheckResourceAttr(mx, "ttl", "20"),
					resource.TestCheckResourceAttr(mx, "hostname", "mail2."+domainName),

					resource.TestCheckResourceAttr(txt, "content", `"test value for TXT record"`),
					resource.TestCheckResourceAttr(txt, "content_normalized", `"test value for TXT record"`),
				),
			},

			// Import domain
			{
				ResourceName:      d,
				ImportState:       true,
				ImportStateVerify: true,
			},

			// Import domain record (only the ID is known; Read() locates the parent domain)
			{
				ResourceName:      mx,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
