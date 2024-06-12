package shopify

import (
	"context"
	"fmt"

	"github.com/gempages/go-shopify-graphql-model/graph/model"
)

type DiscountService interface {
	AutomaticAppCreate(ctx context.Context, discount model.DiscountAutomaticAppInput) (output *model.DiscountAutomaticApp, err error)
	AutomaticAppUpdate(ctx context.Context, discountBaseID string, discount DiscountAutomaticAppInput) (output *model.DiscountAutomaticApp, err error)
}

type DiscountServiceOp struct {
	client *Client
}

var _ DiscountService = &DiscountServiceOp{}

type mutationDiscountAutomaticAppCreate struct {
	DiscountAutomaticCreateAppPayload model.DiscountAutomaticAppCreatePayload `json:"discountAutomaticAppCreate"`
}

type mutationDiscountAutomaticAppUpdate struct {
	DiscountAutomaticAppUpdatePayload model.DiscountAutomaticAppUpdatePayload `json:"discountAutomaticAppUpdate"`
}

type DiscountAutomaticAppInput struct {
	model.DiscountAutomaticAppInput
	ClearEndsAt bool `json:"-"`
}

func (i *DiscountAutomaticAppInput) ToMap() map[string]any {
	result := make(map[string]any)
	if i.ClearEndsAt {
		result["endsAt"] = nil
	}
	if i.EndsAt != nil {
		result["endsAt"] = i.EndsAt
	}
	if i.Title != nil {
		result["title"] = i.Title
	}
	if i.CombinesWith != nil {
		result["combinesWith"] = i.CombinesWith
	}
	if i.Metafields != nil {
		result["metafields"] = i.Metafields
	}
	return result
}

var discountAutomaticAppCreate = `
mutation discountAutomaticAppCreate($automaticAppDiscount: DiscountAutomaticAppInput!) {
  discountAutomaticAppCreate(automaticAppDiscount: $automaticAppDiscount) {
    userErrors {
      field
      message
    }
    automaticAppDiscount {
      discountId
      title
      startsAt
      endsAt
      status
      appDiscountType {
        appKey
        functionId
      }
      combinesWith {
        orderDiscounts
        productDiscounts
        shippingDiscounts
      }
    }
  }
}
`

var discountAutomaticAppUpdate = `
mutation discountAutomaticAppUpdate($automaticAppDiscount: DiscountAutomaticAppInput!, $id: ID!) {
  discountAutomaticAppUpdate(automaticAppDiscount: $automaticAppDiscount, id: $id) {
    automaticAppDiscount {
      discountId
      title
      startsAt
      endsAt
      status
      appDiscountType {
        appKey
        functionId
      }
      combinesWith {
        orderDiscounts
        productDiscounts
        shippingDiscounts
      }
    }
    userErrors {
      field
      message
    }
  }
}
`

func (s *DiscountServiceOp) AutomaticAppCreate(ctx context.Context, input model.DiscountAutomaticAppInput) (output *model.DiscountAutomaticApp, err error) {
	out := mutationDiscountAutomaticAppCreate{}
	vars := map[string]any{
		"automaticAppDiscount": input,
	}

	if err := s.client.gql.MutateString(ctx, discountAutomaticAppCreate, vars, &out); err != nil {
		return nil, fmt.Errorf("gql.MutateString: %w", err)
	}

	if len(out.DiscountAutomaticCreateAppPayload.UserErrors) > 0 {
		return nil, fmt.Errorf("%+v", out.DiscountAutomaticCreateAppPayload.UserErrors)
	}

	return out.DiscountAutomaticCreateAppPayload.AutomaticAppDiscount, nil
}

func (s *DiscountServiceOp) AutomaticAppUpdate(ctx context.Context, discountBaseID string, input DiscountAutomaticAppInput) (output *model.DiscountAutomaticApp, err error) {
	out := mutationDiscountAutomaticAppUpdate{}
	vars := map[string]any{
		"id":                   discountBaseID,
		"automaticAppDiscount": input.ToMap(),
	}
	if err := s.client.gql.MutateString(ctx, discountAutomaticAppUpdate, vars, &out); err != nil {
		return nil, fmt.Errorf("gql.MutateString: %w", err)
	}

	if len(out.DiscountAutomaticAppUpdatePayload.UserErrors) > 0 {
		return nil, fmt.Errorf("%+v", out.DiscountAutomaticAppUpdatePayload.UserErrors)
	}

	return out.DiscountAutomaticAppUpdatePayload.AutomaticAppDiscount, nil
}
