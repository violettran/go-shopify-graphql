package shopify

import (
	"context"
	"fmt"

	"github.com/gempages/go-shopify-graphql-model/graph/model"
)

type AppService interface {
	GetCurrentAppInstallation(ctx context.Context) (*model.AppInstallation, error)
	FindActiveAppSubscriptions(ctx context.Context) ([]model.AppSubscription, error)
	ListAppSubscriptions(ctx context.Context, opts ...QueryOption) (*model.AppSubscriptionConnection, error)
}

type AppServiceOp struct {
	client *Client
}

var _ AppService = &AppServiceOp{}

var queryCurrentAppInstallation = fmt.Sprintf(`
query {
  currentAppInstallation {
	id
	accessScopes {
	  handle
	}
	app {
	  id
	  title
	  embedded
	  isPostPurchaseAppInUse
	  developerType
	}
	activeSubscriptions {
	  createdAt
	  currentPeriodEnd
	  id
	  name
	  returnUrl
	  status
	  test
	  trialDays
	  lineItems {
		id
		%s
	  }
	}
  }
}
`, appSubscriptionLineItemPlan)

func (a *AppServiceOp) GetCurrentAppInstallation(ctx context.Context) (*model.AppInstallation, error) {
	out := struct {
		CurrentAppInstallation *model.AppInstallation `json:"currentAppInstallation"`
	}{}

	err := a.client.gql.QueryString(ctx, queryCurrentAppInstallation, nil, &out)
	if err != nil {
		return nil, err
	}

	return out.CurrentAppInstallation, nil
}

var queryActiveSubscriptions = fmt.Sprintf(`
query {
	currentAppInstallation {
		activeSubscriptions {
			createdAt
			currentPeriodEnd
			id
			name
			returnUrl
			status
			test
			trialDays
		}
	}
}
`)

func (a *AppServiceOp) FindActiveAppSubscriptions(ctx context.Context) ([]model.AppSubscription, error) {
	out := struct {
		CurrentAppInstallation *model.AppInstallation `json:"currentAppInstallation"`
	}{}

	err := a.client.gql.QueryString(ctx, queryActiveSubscriptions, nil, &out)
	if err != nil {
		return nil, err
	}

	return out.CurrentAppInstallation.ActiveSubscriptions, nil
}

var queryAppSubscriptions = fmt.Sprintf(`
query allSubscriptions(
    $first: Int,
    $after: String,
    $reverse: Boolean = false,
    $sortKey: AppSubscriptionSortKeys = CREATED_AT
) {
    allSubscriptions(
        first: $first,
        after: $after,
        reverse: $reverse,
        sortKey: $sortKey
    ) {
        edges {
            node {
				__typename
                createdAt
                currentPeriodEnd
                id
                name
                status
				returnUrl
				test
				trialDays
            }
            cursor
        }
        pageInfo {
            hasNextPage
            hasPreviousPage
			startCursor
      		endCursor
        }
    }
}
`)

func (a *AppServiceOp) ListAppSubscriptions(ctx context.Context, opts ...QueryOption) (*model.AppSubscriptionConnection, error) {
	queryOpt := appServiceQueryOptionBuilder{}
	for _, opt := range opts {
		opt(&queryOpt)
	}
	vars := queryOpt.Build()

	out := model.QueryRoot{}
	err := a.client.gql.QueryString(ctx, queryAppSubscriptions, vars, &out)
	if err != nil {
		return nil, fmt.Errorf("gql.QueryString: %w", err)
	}

	if out.AppInstallation == nil {
		return &model.AppSubscriptionConnection{}, nil
	}

	return out.AppInstallation.AllSubscriptions, nil
}
