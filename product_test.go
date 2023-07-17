package shopify_test

import (
	"os"
	"testing"

	"github.com/gempages/go-shopify-graphql"
	shopifyGraph "github.com/gempages/go-shopify-graphql/graph"
)

var (
	domain = os.Getenv("SHOPIFY_SHOP_DOMAIN")
	token  = os.Getenv("SHOPIFY_API_TOKEN")
)

func TestListProducts(t *testing.T) {
	opts := []shopifyGraph.Option{
		shopifyGraph.WithToken(token),
	}

	c := shopify.NewClientWithOpts(domain, opts...)
	results, err := c.Product.ListAll()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if len(results) == 0 {
		t.Error("results empty")
		t.FailNow()
	}
	if results[0].ID == "" {
		t.Error("ID is empty")
		t.FailNow()
	}
}
