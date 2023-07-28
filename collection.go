package shopify

import (
	"context"
	"fmt"
	"strings"

	"github.com/gempages/go-shopify-graphql-model/graph/model"
	"github.com/gempages/go-shopify-graphql/graphql"
	"github.com/gempages/go-shopify-graphql/utils"
	log "github.com/sirupsen/logrus"
)

type CollectionService interface {
	List(query string) ([]*model.Collection, error)
	ListAll() ([]*model.Collection, error)
	ListByCursor(first int, cursor string) (*model.CollectionConnection, error)
	ListWithFields(first int, cursor string, query string, fields string) (*model.CollectionConnection, error)

	Get(id string) (*model.Collection, error)
	GetSingleCollection(id string, cursor string) (*model.Collection, error)

	Create(collection model.CollectionInput) (string, error)
	CreateBulk(collections []model.CollectionInput) error

	Update(collection model.CollectionInput) error
}

type CollectionServiceOp struct {
	client *Client
}

var _ CollectionService = &CollectionServiceOp{}

type mutationCollectionCreate struct {
	CollectionCreateResult CollectionCreateResult `graphql:"collectionCreate(input: $input)" json:"collectionCreate"`
}

type mutationCollectionUpdate struct {
	CollectionCreateResult CollectionCreateResult `graphql:"collectionUpdate(input: $input)" json:"collectionUpdate"`
}

type CollectionCreateResult struct {
	Collection struct {
		ID string `json:"id,omitempty"`
	}
	UserErrors []UserErrors
}

var collectionQuery = `
	id
	handle
	title

	products(first:250, after: $cursor){
		edges{
			node{
				id
			}
			cursor
		}
		pageInfo{
			hasNextPage
		}
	}
`

var collectionSingleQuery = `
  id
  title
  updatedAt
  handle
  description
  descriptionHtml
  templateSuffix
  productsCount
  seo {
    description
    title
  }
  products(first: 250) {
    edges {
      node {
        id
      }
      cursor
    }
  }
`

var collectionSingleQueryWithCursor = `
  id
  title
  handle
  updatedAt
  description
  descriptionHtml
  templateSuffix
  productsCount
  seo {
    description
    title
  }
  products(first: 250, after: $cursor) {
    edges {
      node {
        id
      }
      cursor
    }
    pageInfo{
      hasNextPage
    }
  }
`

var collectionBulkQuery = `
	id
	handle
	title
	updatedAt
	description
	descriptionHtml
	templateSuffix
	productsCount
	seo{
		description
		title
	}
	image {
		altText
		height
		id
		src
		width
	}
`

var collectionWithProductsBulkQuery = `
	id
	handle
	title
	updatedAt
	description
	descriptionHtml
	templateSuffix
	productsCount
	seo{
		description
		title
	}
	image {
		altText
		height
		id
		src
		width
	}
	products {
		edges {
		  node {
			id
		  }
		  cursor
		}
	}
`

func (s *CollectionServiceOp) List(query string) ([]*model.Collection, error) {
	q := fmt.Sprintf(`
		{
			collections(query: "$query"){
				edges{
					node{
						%s
					}
				}
			}
		}
	`, collectionWithProductsBulkQuery)

	q = strings.ReplaceAll(q, "$query", query)

	res := make([]*model.Collection, 0)
	err := s.client.BulkOperation.BulkQuery(q, &res)
	if err != nil {
		return nil, fmt.Errorf("bulk query: %w", err)
	}

	return res, nil
}

func (s *CollectionServiceOp) ListAll() ([]*model.Collection, error) {
	q := fmt.Sprintf(`
		{
			collections{
				edges{
					node{
						%s
					}
				}
			}
		}
	`, collectionWithProductsBulkQuery)

	res := make([]*model.Collection, 0)
	err := s.client.BulkOperation.BulkQuery(q, &res)
	if err != nil {
		return nil, fmt.Errorf("bulk query: %w", err)
	}

	return res, nil
}

func (s *CollectionServiceOp) ListByCursor(first int, cursor string) (*model.CollectionConnection, error) {
	q := fmt.Sprintf(`
		query collections($first: Int!, $cursor: String) {
			collections(first: $first, after: $cursor){
                edges{
					node {
						%s
					}
                    cursor
                }
                pageInfo {
                      hasNextPage
                }
			}
		}
	`, collectionBulkQuery)

	vars := map[string]interface{}{
		"first": first,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	out := model.QueryRoot{}
	err := utils.ExecWithRetries(s.client.retries, func() error {
		return s.client.gql.QueryString(context.Background(), q, vars, &out)
	})
	if err != nil {
		return nil, err
	}

	return out.Collections, nil
}

func (s *CollectionServiceOp) ListWithFields(first int, cursor, query, fields string) (*model.CollectionConnection, error) {
	if fields == "" {
		fields = `id`
	}

	q := fmt.Sprintf(`
		query collections($first: Int!, $cursor: String, $query: String) {
			collections(first: $first, after: $cursor, query:$query){
				edges{
					cursor
					node {
						%s
					}
				}
                pageInfo {
                      hasNextPage
                }
			}
		}
	`, fields)

	vars := map[string]interface{}{
		"first": first,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}
	if query != "" {
		vars["query"] = query
	}

	out := model.QueryRoot{}
	err := utils.ExecWithRetries(s.client.retries, func() error {
		return s.client.gql.QueryString(context.Background(), q, vars, &out)
	})
	if err != nil {
		return nil, err
	}

	return out.Collections, nil
}

func (s *CollectionServiceOp) Get(id string) (*model.Collection, error) {
	var (
		out *model.Collection
		err error
	)
	out, err = s.getPage(id, "")
	if err != nil {
		return nil, err
	}

	nextPageData := out
	if out != nil && out.Products != nil && out.Products.PageInfo != nil {
		hasNextPage := out.Products.PageInfo.HasNextPage
		for hasNextPage && len(nextPageData.Products.Edges) > 0 {
			cursor := nextPageData.Products.Edges[len(nextPageData.Products.Edges)-1].Cursor
			nextPageData, err = s.getPage(id, cursor)
			if err != nil {
				return nil, err
			}
			out.Products.Edges = append(out.Products.Edges, nextPageData.Products.Edges...)
			hasNextPage = nextPageData.Products.PageInfo.HasNextPage
		}
	}

	return out, nil
}

func (s *CollectionServiceOp) getPage(id graphql.ID, cursor string) (*model.Collection, error) {
	q := fmt.Sprintf(`
		query collection($id: ID!, $cursor: String) {
			collection(id: $id){
				%s
			}
		}
	`, collectionQuery)

	vars := map[string]interface{}{
		"id": id,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	out := model.QueryRoot{}
	err := utils.ExecWithRetries(s.client.retries, func() error {
		return s.client.gql.QueryString(context.Background(), q, vars, &out)
	})
	if err != nil {
		return nil, err
	}

	return out.Collection, nil
}

func (s *CollectionServiceOp) GetSingleCollection(id string, cursor string) (*model.Collection, error) {
	q := ""
	if cursor != "" {
		q = fmt.Sprintf(`
    query collection($id: ID!, $cursor: String) {
      collection(id: $id){
        %s
      }
    }
    `, collectionSingleQueryWithCursor)
	} else {
		q = fmt.Sprintf(`
    query collection($id: ID!) {
      collection(id: $id){
        %s
      }
    }
    `, collectionSingleQuery)
	}

	vars := map[string]interface{}{
		"id": id,
	}
	if cursor != "" {
		vars["cursor"] = cursor
	}

	out := model.QueryRoot{}
	err := utils.ExecWithRetries(s.client.retries, func() error {
		return s.client.gql.QueryString(context.Background(), q, vars, &out)
	})
	if err != nil {
		return nil, err
	}

	return out.Collection, nil
}

func (s *CollectionServiceOp) CreateBulk(collections []model.CollectionInput) error {
	for _, c := range collections {
		_, err := s.client.Collection.Create(c)
		if err != nil {
			log.Warnf("Couldn't create collection (%v): %s", c, err)
		}
	}

	return nil
}

func (s *CollectionServiceOp) Create(collection model.CollectionInput) (string, error) {
	var id string
	m := mutationCollectionCreate{}

	vars := map[string]interface{}{
		"input": collection,
	}
	err := utils.ExecWithRetries(s.client.retries, func() error {
		return s.client.gql.Mutate(context.Background(), &m, vars)
	})
	if err != nil {
		return id, err
	}

	if len(m.CollectionCreateResult.UserErrors) > 0 {
		return id, fmt.Errorf("%+v", m.CollectionCreateResult.UserErrors)
	}

	id = m.CollectionCreateResult.Collection.ID
	return id, nil
}

func (s *CollectionServiceOp) Update(collection model.CollectionInput) error {
	m := mutationCollectionUpdate{}

	vars := map[string]interface{}{
		"input": collection,
	}
	err := utils.ExecWithRetries(s.client.retries, func() error {
		return s.client.gql.Mutate(context.Background(), &m, vars)
	})
	if err != nil {
		return err
	}

	if len(m.CollectionCreateResult.UserErrors) > 0 {
		return fmt.Errorf("%+v", m.CollectionCreateResult.UserErrors)
	}

	return nil
}
