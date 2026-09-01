package handler

import (
	"context"
	"testing"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

func TestCandidateProfileRefreshBuildsReusableUserSnapshot(t *testing.T) {
	originalListUsers := listCandidateProfileUsers
	listCalls := 0
	listCandidateProfileUsers = func() ([]*casdoorsdk.User, error) {
		listCalls++
		return []*casdoorsdk.User{{
			Id:          "casdoor-user-1",
			Name:        "candidate-account",
			DisplayName: "Candidate One",
		}}, nil
	}
	t.Cleanup(func() { listCandidateProfileUsers = originalListUsers })

	cache := NewCandidateProfileCache(&adminReadMidClient{})
	if err := cache.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if listCalls != 1 {
		t.Fatalf("Casdoor list calls = %d, want 1", listCalls)
	}
	if candidateULID, ok := cache.ULIDForUUID("casdoor-user-1"); !ok || candidateULID != "candidate-casdoor-user-1" {
		t.Fatalf("cached candidate mapping = %q, %v", candidateULID, ok)
	}
	if name := cache.NameOrQueue("candidate-casdoor-user-1"); name != "Candidate One" {
		t.Fatalf("cached candidate name = %q", name)
	}

	users, ok := cache.Users()
	if !ok || len(users) != 1 || users[0].Id != "casdoor-user-1" {
		t.Fatalf("cached users = %+v, ready=%v", users, ok)
	}
	mutatedUsers := append(users, &casdoorsdk.User{Id: "caller-only"})
	if len(mutatedUsers) != 2 {
		t.Fatalf("caller mutation length = %d, want 2", len(mutatedUsers))
	}
	secondSnapshot, _ := cache.Users()
	if len(secondSnapshot) != 1 {
		t.Fatalf("caller mutated cached user slice: %+v", secondSnapshot)
	}
}

func TestCandidateProfileUsersOrRefreshInitializesMissingSnapshot(t *testing.T) {
	originalListUsers := listCandidateProfileUsers
	listCandidateProfileUsers = func() ([]*casdoorsdk.User, error) {
		return []*casdoorsdk.User{{Id: "casdoor-user-2", Name: "candidate-two"}}, nil
	}
	t.Cleanup(func() { listCandidateProfileUsers = originalListUsers })

	cache := NewCandidateProfileCache(&adminReadMidClient{})
	users, err := cache.UsersOrRefresh(context.Background())
	if err != nil {
		t.Fatalf("UsersOrRefresh() error = %v", err)
	}
	if len(users) != 1 || users[0].Id != "casdoor-user-2" || !cache.Ready() {
		t.Fatalf("initialized users = %+v, ready=%v", users, cache.Ready())
	}
}
