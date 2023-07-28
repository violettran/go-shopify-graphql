package shopify

import (
	"context"
	"fmt"

	"github.com/gempages/go-shopify-graphql-model/graph/model"
)

type WebhookService interface {
	NewWebhookSubscription(topic model.WebhookSubscriptionTopic, input model.WebhookSubscriptionInput) (output *model.WebhookSubscription, err error)
	NewEventBridgeWebhookSubscription(topic model.WebhookSubscriptionTopic, input model.WebhookSubscriptionInput) (output *model.WebhookSubscription, err error)

	ListWebhookSubscriptions(topics []model.WebhookSubscriptionTopic) (output []*model.WebhookSubscription, err error)
	DeleteWebhook(webhookID string) (deletedID *string, err error)
}

type WebhookServiceOp struct {
	client *Client
}

var _ WebhookService = &WebhookServiceOp{}

type mutationWebhookCreate struct {
	WebhookCreateResult *model.WebhookSubscriptionCreatePayload `graphql:"webhookSubscriptionCreate(topic: $topic, webhookSubscription: $webhookSubscription)" json:"webhookSubscriptionCreate"`
}

type mutationWebhookDelete struct {
	WebhookDeleteResult *model.WebhookSubscriptionDeletePayload `graphql:"webhookSubscriptionDelete(id: $id)" json:"webhookSubscriptionCreate"`
}

type mutationEventBridgeWebhookCreate struct {
	EventBridgeWebhookCreateResult *model.EventBridgeWebhookSubscriptionCreatePayload `graphql:"eventBridgeWebhookSubscriptionCreate(topic: $topic, webhookSubscription: $webhookSubscription)" json:"eventBridgeWebhookSubscriptionCreate"`
}

func (w WebhookServiceOp) NewWebhookSubscription(topic model.WebhookSubscriptionTopic, input model.WebhookSubscriptionInput) (output *model.WebhookSubscription, err error) {
	m := mutationWebhookCreate{}
	vars := map[string]interface{}{
		"topic":               topic,
		"webhookSubscription": input,
	}
	err = w.client.gql.Mutate(context.Background(), &m, vars)
	if err != nil {
		return
	}

	if len(m.WebhookCreateResult.UserErrors) > 0 {
		err = fmt.Errorf("%+v", m.WebhookCreateResult.UserErrors)
		return
	}

	return m.WebhookCreateResult.WebhookSubscription, nil
}

func (w WebhookServiceOp) NewEventBridgeWebhookSubscription(topic model.WebhookSubscriptionTopic, input model.WebhookSubscriptionInput) (output *model.WebhookSubscription, err error) {
	m := mutationEventBridgeWebhookCreate{}
	vars := map[string]interface{}{
		"topic":               topic,
		"webhookSubscription": input,
	}

	err = w.client.gql.Mutate(context.Background(), &m, vars)
	if err != nil {
		return
	}

	if len(m.EventBridgeWebhookCreateResult.UserErrors) > 0 {
		err = fmt.Errorf("%+v", m.EventBridgeWebhookCreateResult.UserErrors)
		return
	}

	return m.EventBridgeWebhookCreateResult.WebhookSubscription, nil
}

func (w WebhookServiceOp) DeleteWebhook(webhookID string) (deletedID *string, err error) {
	m := mutationWebhookDelete{}
	vars := map[string]interface{}{
		"id": webhookID,
	}
	err = w.client.gql.Mutate(context.Background(), &m, vars)
	if err != nil {
		return
	}

	if len(m.WebhookDeleteResult.UserErrors) > 0 {
		err = fmt.Errorf("%+v", m.WebhookDeleteResult.UserErrors)
		return
	}
	return m.WebhookDeleteResult.DeletedWebhookSubscriptionID, nil
}

func (w WebhookServiceOp) ListWebhookSubscriptions(topics []model.WebhookSubscriptionTopic) (output []*model.WebhookSubscription, err error) {
	queryFormat := `query webhookSubscriptions($first: Int!, $topics: [WebhookSubscriptionTopic!]%s) {
		webhookSubscriptions(first: $first, topics: $topics%s) {
		  edges {
			cursor
			node {
			  id,
			  topic,
			  endpoint {
				__typename
				... on WebhookHttpEndpoint {
				  callbackUrl
				}
				... on WebhookEventBridgeEndpoint{
				  arn
				}
			  }
			  callbackUrl
			  format
			  topic
			  includeFields
			  createdAt
			  updatedAt
			}
		  }
		  pageInfo {
			hasNextPage
		  }
		}
	  }`

	var (
		cursor string
		vars   = map[string]interface{}{
			"first":  200,
			"topics": topics,
		}
	)
	for {
		var (
			query string
			out   model.QueryRoot
		)
		if cursor != "" {
			vars["after"] = cursor
			query = fmt.Sprintf(queryFormat, ", $after: String", ", after: $after")
		} else {
			query = fmt.Sprintf(queryFormat, "", "")
		}
		err = w.client.gql.QueryString(context.Background(), query, vars, &out)
		if err != nil {
			return
		}
		for _, wh := range out.WebhookSubscriptions.Edges {
			output = append(output, wh.Node)
		}
		if out.WebhookSubscriptions.PageInfo.HasNextPage {
			cursor = out.WebhookSubscriptions.Edges[len(out.WebhookSubscriptions.Edges)-1].Cursor
		} else {
			break
		}
	}
	return
}
