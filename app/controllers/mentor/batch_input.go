package mentor

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"

	"github.com/JorgeIba/Hurion/app/services"
)

type batchInput[T any] []T

// Entry numbers are one-based; zero identifies the document or whole input.
type batchInputIssue struct {
	entry, line, column int
	err                 error
}

func parseBatchInput[T any](document jsontext.Value) (batchInput[T], []batchInputIssue) {
	var items []jsontext.Value
	// Document syntax was checked by transport; strict entry decoding retains
	// per-entry diagnostics for duplicate names and invalid Unicode.
	if err := jsonv2.Unmarshal(document, &items, jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true)); err != nil {
		return nil, []batchInputIssue{{err: fmt.Errorf("expected one JSON array: %w", err)}}
	}

	if items == nil {
		return nil, []batchInputIssue{{err: errors.New("expected one JSON array, not null")}}
	}

	entries := make(batchInput[T], len(items))
	var issues []batchInputIssue
	for index, item := range items {
		var err error
		if string(item) == "null" {
			err = errors.New("entry must be a JSON object, not null")
		} else {
			err = jsonv2.Unmarshal(item, &entries[index], jsonv2.RejectUnknownMembers(true))
		}
		if err != nil {
			issues = append(issues, batchInputIssue{entry: index + 1, err: err})
		}
	}

	if len(issues) > 0 {
		return nil, issues
	}
	return entries, nil
}

func (input batchInput[T]) validate(validateEntry func(T) error) []batchInputIssue {
	if err := services.ValidateBatchSize(len(input)); err != nil {
		return []batchInputIssue{{err: err}}
	}

	var issues []batchInputIssue
	for index, entry := range input {
		if err := validateEntry(entry); err != nil {
			issues = append(issues, batchInputIssue{entry: index + 1, err: err})
		}
	}
	return issues
}
