package g42cloud

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccCCENodeV3DataSource_basic(t *testing.T) {
	rName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(5))
	resourceName := "data.g42cloud_cce_node.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCCENodeV3DataSource_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCCENodeV3DataSourceID(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
				),
			},
		},
	})
}

func testAccCheckCCENodeV3DataSourceID(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Can't find nodes data source: %s ", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Node data source ID not set ")
		}

		return nil
	}
}

func testAccCCENodeV3DataSource_basic(rName string) string {
	return fmt.Sprintf(`
%s

data "g42cloud_cce_node" "test" {
  cluster_id = g42cloud_cce_cluster.test.id
  name       = g42cloud_cce_node.test.name
}
`, testAccCCENodeV3_basic(rName))
}
