package aws

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appmesh"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func init() {
	resource.AddTestSweepers("aws_appmesh_mesh", &resource.Sweeper{
		Name: "aws_appmesh_mesh",
		F:    testSweepAppmeshMeshes,
		Dependencies: []string{
			"aws_appmesh_virtual_router",
		},
	})
}

func testSweepAppmeshMeshes(region string) error {
	client, err := sharedClientForRegion(region)
	if err != nil {
		return fmt.Errorf("error getting client: %s", err)
	}
	conn := client.(*AWSClient).appmeshconn

	err = conn.ListMeshesPages(&appmesh.ListMeshesInput{}, func(page *appmesh.ListMeshesOutput, isLast bool) bool {
		if page == nil {
			return !isLast
		}

		for _, mesh := range page.Meshes {
			name := aws.StringValue(mesh.MeshName)

			input := &appmesh.DeleteMeshInput{
				MeshName: aws.String(name),
			}

			log.Printf("[INFO] Deleting Appmesh Mesh: %s", name)
			_, err := conn.DeleteMesh(input)

			if err != nil {
				log.Printf("[ERROR] Error deleting Appmesh Mesh (%s): %s", name, err)
			}
		}

		return !isLast
	})
	if err != nil {
		if testSweepSkipSweepError(err) {
			log.Printf("[WARN] Skipping Appmesh Mesh sweep for %s: %s", region, err)
			return nil
		}
		return fmt.Errorf("error retrieving Appmesh Meshes: %s", err)
	}

	return nil
}

func testAccAwsAppmeshMesh_basic(t *testing.T) {
	var mesh appmesh.MeshData
	resourceName := "aws_appmesh_mesh.foo"
	rName := fmt.Sprintf("tf-test-%d", acctest.RandInt())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAppmeshMeshDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAppmeshMeshConfig(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAppmeshMeshExists(
						resourceName, &mesh),
					resource.TestCheckResourceAttr(
						resourceName, "name", rName),
					resource.TestCheckResourceAttrSet(
						resourceName, "created_date"),
					resource.TestCheckResourceAttrSet(
						resourceName, "last_updated_date"),
					resource.TestMatchResourceAttr(
						resourceName, "arn", regexp.MustCompile(fmt.Sprintf("^arn:[^:]+:appmesh:[^:]+:\\d{12}:mesh/%s", rName))),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckAppmeshMeshDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).appmeshconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_appmesh_mesh" {
			continue
		}

		_, err := conn.DescribeMesh(&appmesh.DescribeMeshInput{
			MeshName: aws.String(rs.Primary.Attributes["name"]),
		})
		if isAWSErr(err, "NotFoundException", "") {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("still exist.")
	}

	return nil
}

func testAccCheckAppmeshMeshExists(name string, v *appmesh.MeshData) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).appmeshconn

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		resp, err := conn.DescribeMesh(&appmesh.DescribeMeshInput{
			MeshName: aws.String(rs.Primary.Attributes["name"]),
		})
		if err != nil {
			return err
		}

		*v = *resp.Mesh

		return nil
	}
}

func testAccAppmeshMeshConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_appmesh_mesh" "foo" {
  name = "%s"
}
`, name)
}
