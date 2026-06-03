package services

import "testing"

func TestShouldPublishReply(t *testing.T) {
	if shouldPublishReply("") {
		t.Error("empty character should not publish")
	}
	if shouldPublishReply("   ") {
		t.Error("whitespace-only character should not publish")
	}
	if !shouldPublishReply("dot-matrix") {
		t.Error("known character should publish")
	}
}
