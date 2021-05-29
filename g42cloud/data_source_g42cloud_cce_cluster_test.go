package g42cloud

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccCCEClusterV3DataSource_basic(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(5))
	resourceName := "data.g42cloud_cce_cluster.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCCEClusterV3DataSource_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCCEClusterV3DataSourceID(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "status", "Available"),
					resource.TestCheckResourceAttr(resourceName, "cluster_type", "VirtualMachine"),
				),
			},
		},
	})
}

func testAccCheckCCEClusterV3DataSourceID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find cluster data source: %s ", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("cluster data source ID not set ")
		}

		return nil
	}
}

func testAccCCEClusterV3DataSource_basic(rName string) string {
	return fmt.Sprintf(`
%s

data "g42cloud_cce_cluster" "test" {
  name = g42cloud_cce_cluster.test.name
}
`, testAccCCEClusterV3_basic(rName))
}
