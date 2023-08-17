package webhook_test

import (
	"os"

	"github.com/gempages/go-shopify-graphql"
	"github.com/gempages/go-shopify-graphql-model/graph/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebhookService", func() {
	var (
		shopifyClient *shopify.Client
		domain        string
		token         string
	)

	BeforeEach(func() {
		domain = os.Getenv("SHOPIFY_SHOP_DOMAIN")
		token = os.Getenv("SHOPIFY_API_TOKEN")
		shopifyClient = shopify.NewClientWithToken(token, domain)
	})

	Describe("NewWebhookSubscription", func() {
		It("creates new webhook subscription", func() {
			callbackURL := "https://gempages.xyz/webhook"

			webhooks, err := shopifyClient.Webhook.ListWebhookSubscriptions([]model.WebhookSubscriptionTopic{model.WebhookSubscriptionTopicProductsUpdate})
			Expect(err).NotTo(HaveOccurred())
			for _, webhook := range webhooks {
				if endpoint, ok := webhook.Endpoint.(*model.WebhookHTTPEndpoint); ok && endpoint.CallbackURL == callbackURL {
					_, err = shopifyClient.Webhook.DeleteWebhook(webhook.ID)
					Expect(err).NotTo(HaveOccurred())
				}
			}

			formatJSON := model.WebhookSubscriptionFormatJSON
			webhook, err := shopifyClient.Webhook.NewWebhookSubscription(model.WebhookSubscriptionTopicProductsUpdate, model.WebhookSubscriptionInput{
				CallbackURL: &callbackURL,
				Format:      &formatJSON,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(webhook).NotTo(BeNil())
			endpoint, ok := webhook.Endpoint.(*model.WebhookHTTPEndpoint)
			Expect(ok).To(BeTrue())
			Expect(endpoint.CallbackURL).To(Equal(callbackURL))
			Expect(webhook.ID).NotTo(BeEmpty())

			_, err = shopifyClient.Webhook.DeleteWebhook(webhook.ID)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
