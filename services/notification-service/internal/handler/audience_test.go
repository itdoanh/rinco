package handler

import (
	"encoding/json"
	"testing"
)

// decodeJSON helper so test doesn't import the main json package shadowed
var decodeJSON = json.Unmarshal

func TestBroadcastAudienceParsing(t *testing.T) {
	jsonBody := []byte(`{
		"type": "system",
		"title": "Heads up",
		"body": "Quarterly review tomorrow",
		"priority": "high",
		"channels": ["in_app", "email"],
		"audience": {
			"subtree_root_id": "00000000-0000-0000-0000-000000000001",
			"max_depth": 3
		}
	}`)

	var req broadcastReq
	if err := decodeJSON(jsonBody, &req); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if req.Type != "system" {
		t.Errorf("Type = %q, want system", req.Type)
	}
	if req.Title != "Heads up" {
		t.Errorf("Title = %q, want 'Heads up'", req.Title)
	}
	if req.Priority != "high" {
		t.Errorf("Priority = %q, want high", req.Priority)
	}
	if len(req.Channels) != 2 || req.Channels[0] != "in_app" || req.Channels[1] != "email" {
		t.Errorf("Channels = %v, want [in_app email]", req.Channels)
	}
	if req.Audience.SubtreeRootID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("SubtreeRootID = %q", req.Audience.SubtreeRootID)
	}
	if req.Audience.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3", req.Audience.MaxDepth)
	}
}

func TestBroadcastAudienceByPath(t *testing.T) {
	jsonBody := []byte(`{
		"type": "company",
		"title": "CEO message",
		"body": "Important update",
		"audience": {
			"subtree_root_path": "root.uuid1.uuid2",
			"max_depth": 5
		}
	}`)

	var req broadcastReq
	if err := decodeJSON(jsonBody, &req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.Audience.SubtreeRootPath != "root.uuid1.uuid2" {
		t.Errorf("SubtreeRootPath = %q", req.Audience.SubtreeRootPath)
	}
	if req.Audience.MaxDepth != 5 {
		t.Errorf("MaxDepth = %d, want 5", req.Audience.MaxDepth)
	}
}

func TestBroadcastAudienceDefaults(t *testing.T) {
	jsonBody := []byte(`{"type":"sys","title":"x","body":"y"}`)
	var req broadcastReq
	if err := decodeJSON(jsonBody, &req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.Audience.SubtreeRootID != "" {
		t.Errorf("SubtreeRootID = %q, want empty", req.Audience.SubtreeRootID)
	}
	if req.Audience.SubtreeRootPath != "" {
		t.Errorf("SubtreeRootPath = %q, want empty", req.Audience.SubtreeRootPath)
	}
	if len(req.Audience.UserIDs) != 0 {
		t.Errorf("UserIDs = %v, want empty", req.Audience.UserIDs)
	}
	if req.Audience.MaxDepth != 0 {
		t.Errorf("MaxDepth = %d, want 0", req.Audience.MaxDepth)
	}
}

