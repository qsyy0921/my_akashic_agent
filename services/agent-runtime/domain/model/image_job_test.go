package model_test

import (
	"testing"
	"time"

	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

func TestImageJobStateMachine(t *testing.T) {
	now := time.Date(2026, 5, 30, 1, 0, 0, 0, time.UTC)
	job, err := model.NewImageJob(
		"img_1",
		"request-1",
		model.ChannelRef{
			Kind:             model.ChannelKindQQ,
			AccountID:        "2365524513",
			ConversationID:   "1049511700",
			ConversationType: model.ConversationTypePrivate,
		},
		"1049511700",
		"古装美女",
		model.ImageJobOptions{Provider: "chatgpt-web", Model: "gpt-image", Count: 1},
		2,
		now,
		nil,
	)
	if err != nil {
		t.Fatalf("new image job returned error: %v", err)
	}

	if err := job.MarkRunning(now.Add(time.Second)); err != nil {
		t.Fatalf("mark running returned error: %v", err)
	}
	if job.Status != model.ImageJobStatusRunning || job.Attempts != 1 {
		t.Fatalf("unexpected running state: status=%s attempts=%d", job.Status, job.Attempts)
	}

	err = job.MarkSucceeded([]model.Attachment{{
		Kind: model.AttachmentKindImage,
		URL:  "file:///tmp/result.png",
	}}, now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("mark succeeded returned error: %v", err)
	}
	if job.Status != model.ImageJobStatusSucceeded || len(job.Results) != 1 {
		t.Fatalf("unexpected succeeded state: status=%s results=%d", job.Status, len(job.Results))
	}
}

