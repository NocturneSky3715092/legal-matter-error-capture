package legalflow

import (
	"fmt"
	"strings"
)

type Stage string

const (
	MatterIntake     Stage = "matter_intake"
	SignedDelivery   Stage = "signed_document_delivery"
	DeadlineFollowUp Stage = "deadline_follow_up"
)

type Failure struct {
	EventID   string `json:"event_id"`
	MatterID  string `json:"matter_id"`
	Stage     Stage  `json:"stage"`
	Message   string `json:"message"`
	Exception string `json:"exception"`
}

type Capture struct {
	Title       string
	Message     string
	Level       string
	Fingerprint []string
	Exception   string
	Context     map[string]string
}

func BuildCapture(f Failure) (Capture, error) {
	if strings.TrimSpace(f.EventID) == "" || strings.TrimSpace(f.MatterID) == "" {
		return Capture{}, fmt.Errorf("event_id and matter_id are required")
	}
	switch f.Stage {
	case MatterIntake, SignedDelivery, DeadlineFollowUp:
	default:
		return Capture{}, fmt.Errorf("unknown legal workflow stage %q", f.Stage)
	}
	if strings.TrimSpace(f.Message) == "" || strings.TrimSpace(f.Exception) == "" {
		return Capture{}, fmt.Errorf("message and exception are required")
	}

	return Capture{
		Title:       fmt.Sprintf("%s failed", f.Stage),
		Message:     f.Message,
		Level:       "error",
		Fingerprint: []string{f.MatterID, string(f.Stage)},
		Exception:   f.Exception,
		Context: map[string]string{
			"event_id":  f.EventID,
			"matter_id": f.MatterID,
			"stage":     string(f.Stage),
		},
	}, nil
}
