package legalflow

import (
	"reflect"
	"testing"
)

func TestBuildCaptureGroupsByMatterAndStage(t *testing.T) {
	tests := []struct {
		name string
		in   Failure
		want []string
	}{
		{"intake validation", Failure{"evt-101", "matter-42", MatterIntake, "conflict check failed", "validation stack"}, []string{"matter-42", "matter_intake"}},
		{"signed delivery", Failure{"evt-102", "matter-42", SignedDelivery, "recipient delivery failed", "delivery stack"}, []string{"matter-42", "signed_document_delivery"}},
		{"deadline follow-up", Failure{"evt-103", "matter-77", DeadlineFollowUp, "reminder dispatch failed", "scheduler stack"}, []string{"matter-77", "deadline_follow_up"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildCapture(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Fingerprint, tt.want) {
				t.Fatalf("fingerprint = %v, want %v", got.Fingerprint, tt.want)
			}
		})
	}
}

func TestBuildCaptureRejectsUnknownStage(t *testing.T) {
	_, err := BuildCapture(Failure{"evt-104", "matter-42", Stage("billing"), "failed", "stack"})
	if err == nil {
		t.Fatal("expected unknown stage to be rejected")
	}
}
