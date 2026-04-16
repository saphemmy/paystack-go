package paystack

import (
	"encoding/json"
	"testing"
)

func TestParams_IdempotencyKeyNotInBody(t *testing.T) {
	key := "abc-123"
	p := Params{IdempotencyKey: &key, Metadata: map[string]interface{}{"order": "42"}}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(data); contains(got, "IdempotencyKey") || contains(got, "idempotency") {
		t.Fatalf("IdempotencyKey should be tagged `json:-`, got %s", got)
	}
	if !contains(string(data), `"metadata"`) {
		t.Fatalf("metadata should appear, got %s", data)
	}
}

func TestParams_MetadataOmitEmpty(t *testing.T) {
	p := Params{}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got := string(data); got != "{}" {
		t.Fatalf("empty Params should marshal to {}, got %s", got)
	}
}

func TestListResponse_EmptyDataWithPositiveTotal(t *testing.T) {
	body := []byte(`{"status":true,"message":"ok","data":[],"meta":{"total":42,"skipped":40,"perPage":50,"page":2,"pageCount":1}}`)
	var out ListResponse[map[string]any]
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out.Data) != 0 {
		t.Fatalf("Data length = %d, want 0", len(out.Data))
	}
	if out.Meta.Total != 42 {
		t.Fatalf("Meta.Total = %d, want 42", out.Meta.Total)
	}
	if out.Meta.Page != 2 {
		t.Fatalf("Meta.Page = %d, want 2", out.Meta.Page)
	}
}

func TestListResponse_GenericTyping(t *testing.T) {
	type item struct {
		ID    int    `json:"id"`
		Label string `json:"label"`
	}
	body := []byte(`{"status":true,"message":"ok","data":[{"id":1,"label":"a"},{"id":2,"label":"b"}],"meta":{"total":2,"perPage":50,"page":1,"pageCount":1}}`)
	var out ListResponse[item]
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(out.Data) != 2 {
		t.Fatalf("Data length = %d, want 2", len(out.Data))
	}
	if out.Data[0].ID != 1 || out.Data[1].Label != "b" {
		t.Fatalf("Data mismatch: %+v", out.Data)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
