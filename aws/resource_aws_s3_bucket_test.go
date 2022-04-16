package aws

import (
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccWasabiBucket_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("terraform-provider-wasabi-test-")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckWasabiBucketResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "wasabi_bucket" "this" {
							bucket = "%s"
						}
					`, rName,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("wasabi_bucket.this", "bucket", rName),
					resource.TestCheckResourceAttr("wasabi_bucket.this", "acl", "private"),
				),
			},
		},
	})
}

// testAccCheckExampleResourceDestroy verifies the Widget
// has been destroyed
func testAccCheckWasabiBucketResourceDestroy(s *terraform.State) error {
	// retrieve the connection established in Provider configuration
	conn := testAccProvider.Meta().(*AWSClient).s3conn

	// loop through the resources in state, verifying each widget
	// is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "wasabi_bucket" {
			continue
		}

		input := &s3.HeadBucketInput{
			Bucket: aws.String(rs.Primary.ID),
		}

		log.Printf("[DEBUG] Reading S3 bucket: %s", input)
		_, err := conn.HeadBucket(input)
		if err == nil {
			return fmt.Errorf("Bucket '%s' still exists", rs.Primary.ID)
		}

		var aerr awserr.Error
		if !errors.As(err, &aerr) {
			return fmt.Errorf("Expected error of type 'awserr.Error', got %w", err)
		}

		if aerr.Code() != ErrorCodeNotFound {
			return fmt.Errorf("Expected error code '%s', got '%s'", ErrorCodeNotFound, aerr.Code())
		}
	}

	return nil
}
