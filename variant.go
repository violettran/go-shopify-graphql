package shopify

import (
	"context"
	"fmt"

	"github.com/gempages/go-shopify-graphql-model/graph/model"
)

var productVariantQuery = `
	id
    product {
		id
	}
	createdAt
	updatedAt
	legacyResourceId
	sku
	selectedOptions{
		name
		value
	}
	image {
		altText
		height
		id
		src
		width
	}
	compareAtPrice
	price
	inventoryQuantity
	barcode
	title
	inventoryPolicy
	position
	inventoryItem {
		tracked
	}
	metafields{
		edges{
			node{
				id
				legacyResourceId
				namespace
				key
				value
				type
				ownerType
			}
		}
	}
`

type VariantService interface {
	List(ctx context.Context, opts ...QueryOption) ([]*model.ProductVariant, error)
}

type VariantServiceOp struct {
	client *Client
}

var _ VariantService = &VariantServiceOp{}

func (s *VariantServiceOp) List(ctx context.Context, opts ...QueryOption) ([]*model.ProductVariant, error) {
	b := &bulkQueryBuilder{
		operationName: "productVariants",
		fields:        productVariantQuery,
	}
	for _, opt := range opts {
		opt(b)
	}
	q := b.Build()

	res := make([]*model.ProductVariant, 0)
	err := s.client.BulkOperation.BulkQuery(ctx, q, &res)
	if err != nil {
		return nil, fmt.Errorf("bulk query: %w", err)
	}

	return res, nil
}
