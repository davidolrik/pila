package git

import (
	"encoding/json"
	"testing"
)

func TestMultiMergeTestResult_JSONSerialization_ErrorOmittedWhenEmpty(t *testing.T) {
	result := MultiMergeTestResult{
		OK:            true,
		BranchResults: []MultiMergeTestBranchResult{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// The "error" key should be absent from JSON when Error is empty
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, exists := raw["error"]; exists {
		t.Errorf("expected \"error\" key to be absent from JSON, got: %s", string(data))
	}
}

func TestMultiMergeTestResult_JSONSerialization_ErrorPresentWhenSet(t *testing.T) {
	result := MultiMergeTestResult{
		OK:            false,
		Error:         "something went wrong",
		BranchResults: []MultiMergeTestBranchResult{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// The "error" key should be present with the correct value
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	errorJSON, exists := raw["error"]
	if !exists {
		t.Fatalf("expected \"error\" key to be present in JSON, got: %s", string(data))
	}

	var errorValue string
	if err := json.Unmarshal(errorJSON, &errorValue); err != nil {
		t.Fatalf("json.Unmarshal(error) error = %v", err)
	}
	if errorValue != "something went wrong" {
		t.Errorf("error = %q, want %q", errorValue, "something went wrong")
	}
}

func TestMultiMergeTestBranchResult_JSONSerialization_ErrorOmittedWhenEmpty(t *testing.T) {
	br := MultiMergeTestBranchResult{
		Name:      "feature-a",
		Status:    "clean",
		MergeType: "sequential",
	}

	data, err := json.Marshal(br)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, exists := raw["error"]; exists {
		t.Errorf("expected \"error\" key to be absent from JSON, got: %s", string(data))
	}
}

func TestMultiMergeTestBranchResult_JSONSerialization_ErrorPresentWhenSet(t *testing.T) {
	br := MultiMergeTestBranchResult{
		Name:      "feature-a",
		Status:    "error",
		MergeType: "sequential",
		Error:     "git merge failed: uncommitted changes",
	}

	data, err := json.Marshal(br)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	errorJSON, exists := raw["error"]
	if !exists {
		t.Fatalf("expected \"error\" key to be present in JSON, got: %s", string(data))
	}

	var errorValue string
	if err := json.Unmarshal(errorJSON, &errorValue); err != nil {
		t.Fatalf("json.Unmarshal(error) error = %v", err)
	}
	if errorValue != "git merge failed: uncommitted changes" {
		t.Errorf("error = %q, want %q", errorValue, "git merge failed: uncommitted changes")
	}
}
