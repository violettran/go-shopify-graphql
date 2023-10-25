package shopify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/gempages/go-shopify-graphql-model/graph/model"
	"github.com/gempages/go-shopify-graphql/graphql"
	"github.com/spf13/cast"
)

type FileService interface {
	Upload(ctx context.Context, fileContent []byte, fileName, mimetype string) (*model.GenericFile, error)
	QueryGenericFile(ctx context.Context, fileID string) (*model.GenericFile, error)
}

type FileServiceOp struct {
	client *Client
}

var _ FileService = &FileServiceOp{}

type mutationStagedUploadsCreate struct {
	StagedUploadsCreateResult model.StagedUploadsCreatePayload `graphql:"stagedUploadsCreate(input: $input)" json:"stagedUploadsCreate"`
}

type mutationFileCreate struct {
	FileCreateResult model.FileCreatePayload `graphql:"fileCreate(files: $files)" json:"fileCreate"`
}

const fileFieldName = "file"
const queryGenericFile = `
		query files($query: String!) {
			files(first: 1, query: $query) {
				edges {
					node {
						id
						updatedAt
						fileStatus
						... on GenericFile {
							id
							url
							updatedAt
							originalFileSize
							mimeType
							fileStatus
							fileErrors {
								code
								details
								message
							}
						}
					}
				}
			}
		}
	`

func (s *FileServiceOp) Upload(ctx context.Context, fileContent []byte, fileName, mimetype string) (*model.GenericFile, error) {
	fileSize := len(fileContent)
	stageCreated, err := s.stagedUploadsCreate(cast.ToString(fileSize), fileName, mimetype)
	if err != nil {
		return nil, fmt.Errorf("s.stagedUploadsCreate: %w", err)
	}

	err = s.uploadFileToStage(ctx, fileContent, fileSize, fileName, stageCreated)
	if err != nil {
		return nil, fmt.Errorf("s.uploadFileToStage: %w", err)
	}

	result, err := s.fileCreate(ctx, stageCreated)
	if err != nil {
		return nil, fmt.Errorf("s.fileCreate: %w", err)
	}

	fileInfo, err := s.QueryGenericFile(ctx, result.Files[0].GetID())
	if err != nil {
		return nil, fmt.Errorf("s.QueryGenericFile: %w", err)
	}

	return fileInfo, nil
}

func (s *FileServiceOp) stagedUploadsCreate(fileSize, fileName, mimetype string) (*model.StagedMediaUploadTarget, error) {
	m := mutationStagedUploadsCreate{}
	method := model.StagedUploadHTTPMethodTypePost

	err := s.client.gql.Mutate(context.Background(), &m, map[string]interface{}{
		"input": []model.StagedUploadInput{
			{
				FileSize:   &fileSize,
				Filename:   fileName,
				HTTPMethod: &method,
				MimeType:   mimetype,
				Resource:   model.StagedUploadTargetGenerateUploadResourceFile,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gql.Mutate: %w", err)
	}

	if len(m.StagedUploadsCreateResult.UserErrors) > 0 {
		return nil, fmt.Errorf("%+v", m.StagedUploadsCreateResult.UserErrors)
	}

	return &m.StagedUploadsCreateResult.StagedTargets[0], nil
}

func (s *FileServiceOp) uploadFileToStage(
	ctx context.Context, file []byte, fileSize int, fileName string, stageCreated *model.StagedMediaUploadTarget,
) error {

	// Create a buffer to store the file contents
	fileBuffer := bytes.NewBuffer(file)

	// Create a multipart form and add parameters
	form := &bytes.Buffer{}
	writer := multipart.NewWriter(form)
	defer writer.Close()

	for _, param := range stageCreated.Parameters {
		writer.WriteField(param.Name, param.Value)
	}

	// Add the file to the form
	fileWriter, err := writer.CreateFormFile(fileFieldName, fileName)
	if err != nil {
		return fmt.Errorf("writer.CreateFormFile: %w", err)
	}
	_, err = io.Copy(fileWriter, fileBuffer)
	if err != nil {
		return err
	}

	// Perform the POST request to the temp target
	postTempTargetURL := stageCreated.URL
	postTempTargetHeaders := map[string]string{
		"Content-Type":   writer.FormDataContentType(),
		"Content-Length": cast.ToString(fileSize),
	}

	err = performHTTPPostWithHeaders(ctx, *postTempTargetURL, form, postTempTargetHeaders)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileServiceOp) fileCreate(ctx context.Context, stageCreated *model.StagedMediaUploadTarget) (*model.FileCreatePayload, error) {
	out := mutationFileCreate{
		FileCreateResult: model.FileCreatePayload{
			Files: []model.File{&model.GenericFile{}},
		},
	}

	vars := map[string]interface{}{
		"files": []model.FileCreateInput{
			{
				OriginalSource: *stageCreated.ResourceURL,
			},
		},
	}

	m := `
	mutation fileCreate($files: [FileCreateInput!]!) {
		fileCreate(files: $files) {
			files {
				id
				alt
				fileStatus
			}
			userErrors {
				field
				message
			}
		}
	}
	`

	err := s.client.gql.MutateString(ctx, m, vars, &out)
	if err != nil {
		return nil, err
	}

	if len(out.FileCreateResult.UserErrors) > 0 {
		return nil, fmt.Errorf("%+v", out.FileCreateResult.UserErrors)
	}

	return &out.FileCreateResult, nil
}

func (s *FileServiceOp) QueryGenericFile(ctx context.Context, fileID string) (*model.GenericFile, error) {
	out := struct {
		Files *model.FileConnection `json:"files"`
	}{
		Files: &model.FileConnection{
			Edges: []model.FileEdge{
				{Node: model.File(&model.GenericFile{})},
			},
		},
	}

	vars := map[string]interface{}{
		"query": graphql.String(fileID),
	}
	err := s.client.gql.QueryString(ctx, queryGenericFile, vars, &out)
	if err != nil {
		return nil, fmt.Errorf("gql.QueryString: %w", err)
	}

	if len(out.Files.Edges) <= 0 {
		return nil, fmt.Errorf("file is not found")
	}

	if len(out.Files.Edges[0].Node.GetFileErrors()) > 0 {
		return nil, fmt.Errorf("%+v", out.Files.Edges[0].Node.GetFileErrors())
	}

	return out.Files.Edges[0].Node.(*model.GenericFile), nil
}

func performHTTPPostWithHeaders(ctx context.Context, url string, body io.Reader, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("DefaultClient.Do: %w", err)
	}
	defer resp.Body.Close()

	return nil
}
