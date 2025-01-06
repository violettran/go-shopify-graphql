package shopify

import (
	"github.com/gempages/go-shopify-graphql-model/graph/model"
	"github.com/getsentry/sentry-go"
)

type appServiceQueryOptionBuilder struct {
	first   int
	after   string
	sortKey *model.AppSubscriptionSortKeys
	reverse *bool
}

func (b *appServiceQueryOptionBuilder) SetFields(fields string) {
	return
}

func (b *appServiceQueryOptionBuilder) SetQuery(query string) {
	return
}

func (b *appServiceQueryOptionBuilder) SetFirst(first int) {
	b.first = first
}

func (b *appServiceQueryOptionBuilder) SetAfter(after string) {
	b.after = after
}

func (b *appServiceQueryOptionBuilder) SetSortKey(sortKey string) {
	b.sortKey = sentry.Pointer(model.AppSubscriptionSortKeys(sortKey))
}

func (b *appServiceQueryOptionBuilder) SetReverse(reverse bool) {
	b.reverse = &reverse
}

func (b *appServiceQueryOptionBuilder) Build() map[string]any {
	vars := map[string]any{
		"first":   b.first,
		"after":   b.after,
		"sortKey": b.sortKey,
		"reverse": b.reverse,
	}

	if b.first == 0 {
		vars["first"] = DefaultQueryLimit
	}

	if b.after != "" {
		vars["after"] = b.after
	}

	if b.reverse != nil {
		vars["reverse"] = *b.reverse
	}

	if b.sortKey != nil {
		vars["sortKey"] = *b.sortKey
	}

	return vars
}
