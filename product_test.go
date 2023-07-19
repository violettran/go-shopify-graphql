package shopify_test

import (
	"os"
	"testing"

	"github.com/gempages/go-shopify-graphql"
	shopifyGraph "github.com/gempages/go-shopify-graphql/graph"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProduct(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Product Suite")
}

var _ = Describe("Product", func() {
	var (
		shopifyClient *shopify.Client
		domain        string
		token         string
	)

	BeforeEach(func() {
		domain = os.Getenv("SHOPIFY_SHOP_DOMAIN")
		token = os.Getenv("SHOPIFY_API_TOKEN")
		opts := []shopifyGraph.Option{
			shopifyGraph.WithToken(token),
		}
		shopifyClient = shopify.NewClientWithOpts(domain, opts...)
	})

	Describe("ListAll", func() {
		It("returns all products", func() {
			results, err := shopifyClient.Product.ListAll()
			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeEmpty())
			for i := range results {
				Expect(results[i].ID).NotTo(BeEmpty())
				Expect(results[i].Title).NotTo(BeEmpty())
				Expect(results[i].Handle).NotTo(BeEmpty())
				Expect(results[i].Images).NotTo(BeNil())
				Expect(results[i].Media).NotTo(BeNil())
				Expect(results[i].Variants).NotTo(BeNil())
				Expect(results[i].CreatedAt).NotTo(BeEmpty())
			}
		})
	})

	Describe("ListWithFields", func() {
		It("returns only requested fields", func() {
			fields := `id title handle`
			first := 1
			results, err := shopifyClient.Product.ListWithFields("", fields, first, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(results).NotTo(BeNil())
			for _, e := range results.Edges {
				Expect(e.Node.ID).NotTo(BeEmpty())
				Expect(e.Node.Title).NotTo(BeEmpty())
				Expect(e.Node.Handle).NotTo(BeEmpty())
				// Other fields should be empty
				Expect(e.Node.Images).To(BeNil())
				Expect(e.Node.Media).To(BeNil())
				Expect(e.Node.Variants).To(BeNil())
			}
		})

		When("query first 5 products", func() {
			It("returns 5 products", func() {
				fields := `id`
				first := 5
				results, err := shopifyClient.Product.ListWithFields("", fields, first, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(results).NotTo(BeNil())
				Expect(len(results.Edges)).To(Equal(first))
			})
		})
	})

	Describe("Get", func() {
		When("ID does not exist", func() {
			It("returns nil without any error", func() {
				product, err := shopifyClient.Product.Get("gid://shopify/Product/0000")
				Expect(err).NotTo(HaveOccurred())
				Expect(product).To(BeNil())
			})
		})

		When("ID exists", func() {
			It("returns the correct product with all of its variants", func() {
				productID := "gid://shopify/Product/8427241144634"
				product, err := shopifyClient.Product.Get(productID)
				Expect(err).NotTo(HaveOccurred())
				Expect(product).NotTo(BeNil())
				Expect(product.ID).To(Equal(productID))
				Expect(len(product.Variants.Edges)).To(Equal(72))
			})
		})
	})

	Describe("GetWithFields", func() {
		When("ID does not exist", func() {
			It("returns nil without any error", func() {
				product, err := shopifyClient.Product.GetWithFields("gid://shopify/Product/0000", "id")
				Expect(err).NotTo(HaveOccurred())
				Expect(product).To(BeNil())
			})
		})

		When("ID exists", func() {
			It("returns the correct product with requested fields", func() {
				productID := "gid://shopify/Product/8427241144634"
				fields := `
					id
					media(first: 100) {
						edges {
							node {
								mediaContentType
								...on MediaImage {
									id
									alt
									mimeType
									image {
										height
										src
										width
									}
								}
								...on Model3d {
									id
									alt
									originalSource {
										url
										format
										filesize
										mimeType
									}
									preview {
										image {
											src
										}
									}
								}
								...on Video {
									id
									alt
									duration
									originalSource {
										url
										format
										mimeType
										height
										width
									}
									preview {
										image {
											src
										}
									}
								}
								...on ExternalVideo {
									id
									originUrl
									embedUrl
									preview {
										image {
											src
										}
									}
								}
							}
							cursor
						}
						pageInfo {
							hasNextPage
						}
					}
				`
				product, err := shopifyClient.Product.GetWithFields(productID, fields)
				Expect(err).NotTo(HaveOccurred())
				Expect(product).NotTo(BeNil())
				Expect(product.ID).To(Equal(productID))
				Expect(len(product.Media.Edges)).To(Equal(6))
				for i := range product.Media.Edges {
					Expect(product.Media.Edges[i].Node).NotTo(BeNil())
				}
			})
		})
	})
})
