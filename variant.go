package shopify

import (
	"context"
	"fmt"

	"github.com/gempages/go-shopify-graphql-model/graph/model"
)

type VariantService interface {
	Update(variant model.ProductVariantInput) error
}

type VariantServiceOp struct {
	client *Client
}

var _ VariantService = &VariantServiceOp{}

type mutationProductVariantUpdate struct {
	ProductVariantUpdateResult productVariantUpdateResult `graphql:"productVariantUpdate(input: $input)" json:"productVariantUpdate"`
}

type productVariantUpdateResult struct {
	UserErrors []UserErrors
}

func (s *VariantServiceOp) Update(variant model.ProductVariantInput) error {
	m := mutationProductVariantUpdate{}

	vars := map[string]interface{}{
		"input": variant,
	}
	err := s.client.gql.Mutate(context.Background(), &m, vars)
	if err != nil {
		return err
	}

	if len(m.ProductVariantUpdateResult.UserErrors) > 0 {
		return fmt.Errorf("%+v", m.ProductVariantUpdateResult.UserErrors)
	}

	return nil
}
