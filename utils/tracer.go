package utils

import (
	"encoding/json"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

func GetDescriptionFromQuery(query string) string {
	var description string
	queryDoc, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil {
		description = ""
	} else {
		for _, op := range queryDoc.Operations {
			// Get all selection in an operation
			for _, selectionSet := range op.SelectionSet {
				field := &ast.Field{}
				data, err := json.Marshal(selectionSet)
				if err != nil {
					description = ""
					break
				}
				err = json.Unmarshal(data, field)
				if err != nil {
					description = ""
				}
			}
		}
		description = strings.Trim(description, ",")
	}
	return description
}
