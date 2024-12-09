package shopify

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/gempages/go-helper/tracing"
	"github.com/gempages/go-shopify-graphql-model/graph/model"
	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/guregu/null.v4"

	"github.com/gempages/go-shopify-graphql/graphql"
	"github.com/gempages/go-shopify-graphql/utils"
)

const (
	edgesFieldName = "Edges"
	nodeFieldName  = "Node"
)

type BulkOperationService interface {
	BulkQuery(ctx context.Context, query string, v interface{}) error

	PostBulkQuery(ctx context.Context, query string) (*string, error)
	GetCurrentBulkQuery(ctx context.Context) (*model.BulkOperation, error)
	GetCurrentBulkQueryResultURL(ctx context.Context) (*string, error)
	WaitForCurrentBulkQuery(ctx context.Context, interval time.Duration) (*model.BulkOperation, error)
	ShouldGetBulkQueryResultURL(ctx context.Context, id *string) (*string, error)
	CancelRunningBulkQuery(ctx context.Context) error
	GetBulkQueryResult(ctx context.Context, id graphql.ID) (*model.BulkOperation, error)
}

type BulkOperationServiceOp struct {
	client *Client
}

var _ BulkOperationService = &BulkOperationServiceOp{}

type mutationBulkOperationRunQuery struct {
	BulkOperationRunQueryResult model.BulkOperationRunQueryPayload `graphql:"bulkOperationRunQuery(query: $query)" json:"bulkOperationRunQuery"`
}

type mutationBulkOperationRunQueryCancel struct {
	BulkOperationCancelResult model.BulkOperationCancelPayload `graphql:"bulkOperationCancel(id: $id)" json:"bulkOperationCancel"`
}

var gidRegex *regexp.Regexp

func init() {
	gidRegex = regexp.MustCompile(`^gid://shopify/(\w+)/\d+$`)
}

func (s *BulkOperationServiceOp) PostBulkQuery(ctx context.Context, query string) (*string, error) {
	m := mutationBulkOperationRunQuery{}
	vars := map[string]interface{}{
		"query": null.StringFrom(query),
	}

	err := s.client.gql.Mutate(ctx, &m, vars)
	if err != nil {
		return nil, fmt.Errorf("error posting bulk query: %w", err)
	}
	if len(m.BulkOperationRunQueryResult.UserErrors) > 0 {
		userErrors, _ := json.MarshalIndent(m.BulkOperationRunQueryResult.UserErrors, "", "    ")
		return nil, fmt.Errorf("error posting bulk query: %s", userErrors)
	}

	return &m.BulkOperationRunQueryResult.BulkOperation.ID, nil
}

func (s *BulkOperationServiceOp) GetCurrentBulkQuery(ctx context.Context) (*model.BulkOperation, error) {
	var q struct {
		CurrentBulkOperation struct {
			model.BulkOperation
		}
	}
	err := s.client.gql.Query(ctx, &q, nil)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return &q.CurrentBulkOperation.BulkOperation, nil
}

func (s *BulkOperationServiceOp) GetCurrentBulkQueryResultURL(ctx context.Context) (*string, error) {
	return s.ShouldGetBulkQueryResultURL(ctx, nil)
}

func (s *BulkOperationServiceOp) ShouldGetBulkQueryResultURL(ctx context.Context, id *string) (*string, error) {
	q, err := s.GetCurrentBulkQuery(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting current bulk operation: %w", err)
	}

	if id != nil && q.ID != *id {
		return nil, fmt.Errorf("bulk operation ID doesn't match, got=%v, want=%v", q.ID, id)
	}

	q, err = s.WaitForCurrentBulkQuery(ctx, 1*time.Second)
	if err != nil {
		return nil, fmt.Errorf("waiting for current bulk operation: %w", err)
	}
	if q.Status != model.BulkOperationStatusCompleted {
		return nil, fmt.Errorf("bulk operation didn't complete, status=%s, error_code=%s", q.Status, q.ErrorCode)
	}

	if q.ErrorCode != nil && q.ErrorCode.String() != "" {
		return nil, fmt.Errorf("bulk operation error: %s", q.ErrorCode)
	}

	if q.ObjectCount == "0" {
		return nil, nil
	}

	if q.URL == nil {
		return nil, fmt.Errorf("empty URL result")
	}

	return q.URL, nil
}

func (s *BulkOperationServiceOp) WaitForCurrentBulkQuery(ctx context.Context, interval time.Duration) (*model.BulkOperation, error) {
	q, err := s.GetCurrentBulkQuery(ctx)
	if err != nil {
		return q, fmt.Errorf("get current bulk query: %w", err)
	}

	for q.Status == model.BulkOperationStatusCreated || q.Status == model.BulkOperationStatusRunning || q.Status == model.BulkOperationStatusCanceling {
		log.Debugf("Bulk operation is still %s...", q.Status)
		span := sentry.StartSpan(ctx, "time.sleep")
		span.Description = "interval"
		time.Sleep(interval)
		tracing.FinishSpan(span, ctx.Err())
		ctx = span.Context()

		q, err = s.GetCurrentBulkQuery(ctx)
		if err != nil {
			return q, fmt.Errorf("get current bulk query continously: %w", err)
		}
	}
	log.Debugf("Bulk operation ready, latest status=%s", q.Status)

	return q, nil
}

func (s *BulkOperationServiceOp) CancelRunningBulkQuery(ctx context.Context) error {
	q, err := s.GetCurrentBulkQuery(ctx)
	if err != nil {
		return err
	}

	if q.Status == model.BulkOperationStatusCreated || q.Status == model.BulkOperationStatusRunning {
		log.Debugln("Canceling running operation")
		operationID := q.ID

		m := mutationBulkOperationRunQueryCancel{}
		vars := map[string]interface{}{
			"id": operationID,
		}

		err = s.client.gql.Mutate(ctx, &m, vars)
		if err != nil {
			return fmt.Errorf("mutation: %w", err)
		}
		if len(m.BulkOperationCancelResult.UserErrors) > 0 {
			return fmt.Errorf("%+v", m.BulkOperationCancelResult.UserErrors)
		}

		q, err = s.GetCurrentBulkQuery(ctx)
		if err != nil {
			return err
		}
		for q.Status == model.BulkOperationStatusCreated || q.Status == model.BulkOperationStatusRunning || q.Status == model.BulkOperationStatusCanceling {
			log.Tracef("Bulk operation still %s...", q.Status)
			q, err = s.GetCurrentBulkQuery(ctx)
			if err != nil {
				return fmt.Errorf("get current bulk query: %w", err)
			}
		}
		log.Debugln("Bulk operation cancelled")
	}

	return nil
}

func (s *BulkOperationServiceOp) BulkQuery(ctx context.Context, query string, out interface{}) error {
	var (
		id  *string
		err error
	)

	// sentry tracing
	span := sentry.StartSpan(ctx, "shopify_graphql.bulk_query")
	span.Data = map[string]interface{}{
		"GraphQL Query": query,
	}
	defer func() {
		tracing.FinishSpan(span, err)
	}()
	ctx = span.Context()
	// end sentry tracing

	_, err = s.WaitForCurrentBulkQuery(ctx, time.Second)
	if err != nil {
		return fmt.Errorf("wait for current bulk query: %w", err)
	}

	id, err = s.PostBulkQuery(ctx, query)
	if err != nil {
		return fmt.Errorf("post bulk query: %w", err)
	}

	if id == nil {
		return fmt.Errorf("posted operation ID is nil")
	}

	url, err := s.ShouldGetBulkQueryResultURL(ctx, id)
	if err != nil {
		return fmt.Errorf("get bulk query result URL: %w", err)
	}

	if url == nil || *url == "" {
		// Empty result
		return nil
	}

	resultFile, err := os.CreateTemp("", "*.jsonl")
	if err != nil {
		return fmt.Errorf("create tempfile: %w", err)
	}
	// Clean up to avoid storage build up
	defer func() {
		_ = resultFile.Close()
		_ = os.Remove(resultFile.Name())
	}()

	err = utils.DownloadFile(ctx, resultFile, *url)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}

	// Prepare file for reading
	_, err = resultFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("seeking result file: %w", err)
	}

	err = parseBulkQueryResult(resultFile, out)
	if err != nil {
		return fmt.Errorf("parse bulk query result: %w", err)
	}

	return nil
}

// GetBulkQueryResult get current status of bulk query id
func (s *BulkOperationServiceOp) GetBulkQueryResult(ctx context.Context, id graphql.ID) (*model.BulkOperation, error) {
	q, err := s.GetCurrentBulkQuery(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current bulk query: %w", err)
	}

	if id != nil && q.ID != id {
		err = fmt.Errorf("bulk operation ID doesn't match, got=%v, want=%v", q.ID, id)
		return q, err
	}
	return q, nil
}

type bulkQueryBuilder struct {
	operationName string
	fields        string
	query         *string
	first         int
	after         string
	sortKey       string
	reverse       *bool
}

func (b *bulkQueryBuilder) SetFields(fields string) {
	b.fields = fields
}

func (b *bulkQueryBuilder) SetQuery(query string) {
	b.query = &query
}

func (b *bulkQueryBuilder) SetFirst(first int) {
	b.first = first
}

func (b *bulkQueryBuilder) SetAfter(after string) {
	b.after = after
}

func (b *bulkQueryBuilder) SetSortKey(sortKey string) {
	b.sortKey = sortKey
}

func (b *bulkQueryBuilder) SetReverse(reverse bool) {
	b.reverse = &reverse
}

func (b *bulkQueryBuilder) Build() string {
	var (
		q       = strings.ReplaceAll(`query $operation { $operation`, "$operation", b.operationName)
		vars    = make([]string, 0)
		varsStr string
	)
	if b.query != nil {
		vars = append(vars, fmt.Sprintf(`query: "%s"`, *b.query))
	}
	if b.sortKey != "" {
		vars = append(vars, fmt.Sprintf(`sortKey: %s`, b.sortKey))
	}
	if b.reverse != nil {
		vars = append(vars, fmt.Sprintf(`reverse: %v`, *b.reverse))
	}
	if len(vars) > 0 {
		varsStr = "(" + strings.Join(vars, ", ") + ")"
	}
	q = fmt.Sprintf(`%s%s {
	edges {
		node {
			%s
		}
	}
}}`, q, varsStr, b.fields)
	return q
}

func parseBulkQueryResult(resultFile *os.File, out interface{}) error {
	var err error
	if reflect.TypeOf(out).Kind() != reflect.Ptr {
		return fmt.Errorf("the out arg is not a pointer")
	}

	outValue := reflect.ValueOf(out)
	outSlice := outValue.Elem()
	if outSlice.Kind() != reflect.Slice {
		return fmt.Errorf("the out arg is not a pointer to a slice interface")
	}

	sliceItemType := outSlice.Type().Elem() // slice item type
	sliceItemKind := sliceItemType.Kind()
	itemType := sliceItemType // slice item underlying type
	if sliceItemKind == reflect.Ptr {
		itemType = itemType.Elem()
	}

	reader := bufio.NewReader(resultFile)
	json := jsoniter.ConfigFastest

	connectionSink := make(map[string]interface{})

	for {
		var line []byte
		line, err = reader.ReadBytes('\n')
		if err != nil {
			break
		}

		parentIDNode := json.Get(line, "__parentId")
		if parentIDNode.LastError() == nil {
			parentID := parentIDNode.ToString()

			gid := json.Get(line, "id")
			if gid.LastError() != nil {
				return fmt.Errorf("The connection type must query the `id` field")
			}
			edgeType, nodeType, connectionFieldName, err := concludeObjectType(gid.ToString())
			if err != nil {
				return err
			}
			node := reflect.New(nodeType).Interface()
			err = json.Unmarshal(line, &node)
			if err != nil {
				return fmt.Errorf("unmarshalling: %w", err)
			}
			nodeVal := reflect.ValueOf(node).Elem()

			var edge interface{}
			var edgeVal reflect.Value
			var nodeField reflect.Value
			if edgeType.Kind() == reflect.Ptr {
				edge = reflect.New(edgeType.Elem()).Interface()
				nodeField = reflect.ValueOf(edge).Elem().FieldByName(nodeFieldName)
				edgeVal = reflect.ValueOf(edge)
			} else {
				edge = reflect.New(edgeType).Interface()

				if reflect.ValueOf(edge).Kind() == reflect.Ptr {
					nodeField = reflect.ValueOf(edge).Elem().FieldByName(nodeFieldName)
				} else {
					nodeField = reflect.ValueOf(edge).FieldByName(nodeFieldName)
				}

				edgeVal = reflect.ValueOf(edge).Elem()
			}

			if !nodeField.IsValid() {
				return fmt.Errorf("Edge in the '%s' doesn't have the Node field", connectionFieldName)
			}
			nodeField.Set(nodeVal)

			var edgesSlice reflect.Value
			var edges map[string]interface{}
			if val, ok := connectionSink[parentID]; ok {
				var ok2 bool
				if edges, ok2 = val.(map[string]interface{}); !ok2 {
					return fmt.Errorf("The connection sink for parent ID '%s' is not a map", parentID)
				}
			} else {
				edges = make(map[string]interface{})
			}

			if val, ok := edges[connectionFieldName]; ok {
				edgesSlice = reflect.ValueOf(val)
			} else {
				edgesSliceCap := 50
				edgesSlice = reflect.MakeSlice(reflect.SliceOf(edgeType), 0, edgesSliceCap)
			}

			edgesSlice = reflect.Append(edgesSlice, edgeVal)

			edges[connectionFieldName] = edgesSlice.Interface()
			connectionSink[parentID] = edges

			continue
		}

		item := reflect.New(itemType).Interface()
		err = json.Unmarshal(line, &item)
		if err != nil {
			return fmt.Errorf("unmarshalling: %w", err)
		}
		itemVal := reflect.ValueOf(item)

		if sliceItemKind == reflect.Ptr {
			outSlice.Set(reflect.Append(outSlice, itemVal))
		} else {
			outSlice.Set(reflect.Append(outSlice, itemVal.Elem()))
		}
	}

	if len(connectionSink) > 0 {
		err := attachNestedConnections(connectionSink, outSlice)
		if err != nil {
			return fmt.Errorf("error processing nested connections: %w", err)
		}
	}

	// check if ReadBytes returned an error different from EOF
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("reading the result file: %w", err)
	}

	return nil
}

func attachNestedConnections(connectionSink map[string]interface{}, outSlice reflect.Value) error {
	for i := 0; i < outSlice.Len(); i++ {
		parent := outSlice.Index(i)
		if parent.Kind() == reflect.Ptr {
			parent = parent.Elem()
		}

		nodeField := parent.FieldByName("Node")
		if nodeField != (reflect.Value{}) {
			if nodeField.Kind() == reflect.Ptr {
				parent = nodeField.Elem()
			} else if nodeField.Kind() == reflect.Interface {
				parent = nodeField.Elem().Elem()
			} else {
				parent = nodeField
			}
		}

		parentIDField := parent.FieldByName("ID")
		if parentIDField == (reflect.Value{}) {
			return fmt.Errorf("No ID field on the first level")
		}
		if parentIDField.Kind() == reflect.Ptr {
			parentIDField = parentIDField.Elem()
		}

		var parentID string
		var ok bool
		if parentID, ok = parentIDField.Interface().(string); !ok {
			return fmt.Errorf("ID field on the first level is not a string")
		}

		var connection interface{}
		if connection, ok = connectionSink[parentID]; !ok {
			continue
		}

		edgeMap := reflect.ValueOf(connection)
		iter := edgeMap.MapRange()
		for iter.Next() {
			connectionName := iter.Key()
			connectionField := parent.FieldByName(connectionName.String())
			if !connectionField.IsValid() {
				return fmt.Errorf("Connection '%s' is not defined on the parent type %s", connectionName.String(), parent.Type().String())
			}

			var connectionValue reflect.Value
			var edgesField reflect.Value
			if connectionField.Kind() == reflect.Ptr {
				connectionValue = reflect.ValueOf(reflect.New(connectionField.Type().Elem()).Interface())
				edgesField = connectionValue.Elem().FieldByName(edgesFieldName)
			} else {
				connectionValue = reflect.ValueOf(reflect.New(connectionField.Type()).Interface())
				edgesField = connectionValue.Elem().FieldByName(edgesFieldName)
			}

			if !edgesField.IsValid() {
				return fmt.Errorf("Connection %s in the '%s' doesn't have the Edges field", connectionName.String(), parent.Type().String())
			}

			edges := reflect.ValueOf(iter.Value().Interface())
			edgesField.Set(edges)

			connectionField.Set(connectionValue)

			err := attachNestedConnections(connectionSink, iter.Value().Elem())
			if err != nil {
				return fmt.Errorf("error attacing a nested connection: %w", err)
			}
		}
	}

	return nil
}

func concludeObjectType(gid string) (reflect.Type, reflect.Type, string, error) {
	submatches := gidRegex.FindStringSubmatch(gid)
	if len(submatches) != 2 {
		return reflect.TypeOf(nil), reflect.TypeOf(nil), "", fmt.Errorf("malformed gid=`%s`", gid)
	}
	resource := submatches[1]
	switch resource {
	case "LineItem":
		return reflect.TypeOf(model.LineItemEdge{}), reflect.TypeOf(&model.LineItem{}), fmt.Sprintf("%ss", resource), nil
	case "FulfillmentOrderLineItem":
		return reflect.TypeOf(model.FulfillmentOrderLineItemEdge{}), reflect.TypeOf(&model.FulfillmentOrderLineItem{}), "LineItems", nil
	case "FulfillmentOrder":
		return reflect.TypeOf(model.FulfillmentOrderEdge{}), reflect.TypeOf(&model.FulfillmentOrder{}), fmt.Sprintf("%ss", resource), nil
	case "MediaImage":
		return reflect.TypeOf(model.MediaEdge{}), reflect.TypeOf(&model.MediaImage{}), "Media", nil
	case "Metafield":
		return reflect.TypeOf(model.MetafieldEdge{}), reflect.TypeOf(&model.Metafield{}), fmt.Sprintf("%ss", resource), nil
	case "Order":
		return reflect.TypeOf(model.OrderEdge{}), reflect.TypeOf(&model.Order{}), fmt.Sprintf("%ss", resource), nil
	case "Product":
		return reflect.TypeOf(model.ProductEdge{}), reflect.TypeOf(&model.Product{}), fmt.Sprintf("%ss", resource), nil
	case "ProductVariant":
		return reflect.TypeOf(model.ProductVariantEdge{}), reflect.TypeOf(&model.ProductVariant{}), "Variants", nil
	case "Collection":
		return reflect.TypeOf(model.CollectionEdge{}), reflect.TypeOf(&model.Collection{}), "Collections", nil
	case "ProductImage":
		return reflect.TypeOf(model.ImageEdge{}), reflect.TypeOf(&model.Image{}), "Images", nil
	case "Video":
		return reflect.TypeOf(model.MediaEdge{}), reflect.TypeOf(&model.Video{}), "Media", nil
	case "Model3d":
		return reflect.TypeOf(model.MediaEdge{}), reflect.TypeOf(&model.Model3d{}), "Media", nil
	case "ExternalVideo":
		return reflect.TypeOf(model.MediaEdge{}), reflect.TypeOf(&model.ExternalVideo{}), "Media", nil
	default:
		return reflect.TypeOf(nil), reflect.TypeOf(nil), "", fmt.Errorf("`%s` not implemented type", resource)
	}
}
