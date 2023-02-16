package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	cluster1Name = "test-cluster-1"
	cluster2Name = "test-cluster-2"
)

func TestAccSKSDataSources(t *testing.T) {
	type testCase struct {
		Config               string
		DataSourceIdentifier string
		DataSourceName       string
		Attributes           testAttrs
	}

	zone := "ch-gva-2"
	dsId := dsSKSClusterIdentifier
	dsName := "my_cluster_ds"
	testCases := []testCase{
		{
			Config: fmt.Sprintf(`
				data %q %q {
				  zone = %q
				  name = %q
				}
				`, dsId, dsName, zone, cluster1Name),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"name": validateString(cluster1Name),
			},
		},
		{
			Config: fmt.Sprintf(`
		data %q %q {
		  zone = %q
		  id = "7149e9fc-75f5-48e6-b9ce-fcdf10f40b12"
		}
		`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"name": validateString(cluster1Name),
			},
		},
	}

	nodepoolName := "my-sks-nodepool"
	dsId = dsSKSNodepoolIdentifier
	dsName = "my_nodepool_ds"
	testCases = append(testCases, []testCase{
		{
			// TODO use cluster id
			Config: fmt.Sprintf(`
data %q %q {
  zone = %q
  cluster_id = "7149e9fc-75f5-48e6-b9ce-fcdf10f40b12"
  name = %q
}
`, dsId, dsName, zone, nodepoolName),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"name": validateString(nodepoolName),
			},
		},
		{
			// TODO use id
			Config: fmt.Sprintf(`
data %q %q {
  zone = %q
  cluster_id = "7149e9fc-75f5-48e6-b9ce-fcdf10f40b12"
  id = "4f6912d5-f761-4e8b-80f3-53e2c4fd0d1f"
}
`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"name": validateString(nodepoolName),
			},
		},
	}...,
	)

	dsId = dsSKSClustersListIdentifier
	dsName = "my_cluster_list"
	testCases = append(testCases, []testCase{
		{
			// TODO use cluster id
			Config: fmt.Sprintf(`
data %q %q {
  zone = %q
  #cluster_id = "7149e9fc-75f5-48e6-b9ce-fcdf10f40b12"
  name = %q
}
`, dsId, dsName, zone, cluster1Name),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"clusters.#":      validateString("1"),
				"clusters.0.name": validateString(cluster1Name),
			},
		},
		{
			Config: fmt.Sprintf(`
data %q %q {
  zone = %q
  labels = {
    "customer" = "/.*telecom.*/"
}
}
`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"clusters.#": validateString("2"),
			},
		},
	}...,
	)

	dsId = dsSKSNodepoolsListIdentifier
	dsName = "my_nodepool_list"
	testCases = append(testCases, []testCase{
		{
			Config: fmt.Sprintf(`
data %q %q {
  zone = %q
  size = 3
  name = "/.*nodepool-2/"
}
`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"nodepools.#":      validateString("1"),
				"nodepools.0.name": validateString("my-sks-nodepool-2"),
			},
		},
		{
			Config: fmt.Sprintf(`
data %q %q {
  zone = %q
  name = "/.*-nodepool.*/"
}
`, dsId, dsName, zone),
			DataSourceIdentifier: dsId,
			DataSourceName:       dsName,
			Attributes: testAttrs{
				"nodepools.#": validateString("2"),
			},
		},
	}...)

	resTC := resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps:             []resource.TestStep{
			// {
			// 	Config:             testAccDataSourceComputeInstanceListConfig,
			// 	ExpectNonEmptyPlan: true,
			// },
		},
	}

	for _, c := range testCases {
		resTC.Steps = append(resTC.Steps, resource.TestStep{
			Config: c.Config,
			Check: resource.ComposeTestCheckFunc(
				testAccSKSDataSourcesAttributes("data."+c.DataSourceIdentifier+"."+c.DataSourceName, c.Attributes)),
		})
	}

	resource.Test(t, resTC)
}

func testAccSKSDataSourcesAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("data source not found in the state")
	}
}
