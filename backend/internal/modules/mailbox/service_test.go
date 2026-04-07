package mailbox

import (
	"context"
	"errors"
	"testing"

	"shiro-email/backend/internal/modules/domain"
)

func TestCreateMailboxRejectsUnverifiedSubdomain(t *testing.T) {
	t.Parallel()

	domainRepo := domain.NewMemoryRepository(nil)
	mailboxRepo := NewMemoryRepository()
	service := NewService(mailboxRepo, domainRepo)
	userID := uint64(7)

	root, err := domainRepo.Upsert(context.Background(), domain.Domain{
		Domain:            "example.com",
		OwnerUserID:       &userID,
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		ProviderAccountID: pointerUint64(11),
		VerificationScore: 100,
		HealthStatus:      "healthy",
		Weight:            100,
	})
	if err != nil {
		t.Fatalf("seed root domain: %v", err)
	}

	child, err := domainRepo.Upsert(context.Background(), domain.Domain{
		Domain:            "relay.example.com",
		OwnerUserID:       &userID,
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		ProviderAccountID: root.ProviderAccountID,
		VerificationScore: 0,
		HealthStatus:      "unknown",
		Weight:            90,
	})
	if err != nil {
		t.Fatalf("seed subdomain: %v", err)
	}

	_, err = service.CreateMailbox(context.Background(), userID, CreateMailboxRequest{
		DomainID:        child.ID,
		LocalPart:       "testbox",
		ExpiresInHours:  24,
	})
	if !errors.Is(err, ErrDomainVerificationRequired) {
		t.Fatalf("expected ErrDomainVerificationRequired, got %v", err)
	}
}

func TestBuildDashboardHidesUnverifiedSubdomainsFromAvailableDomains(t *testing.T) {
	t.Parallel()

	domainRepo := domain.NewMemoryRepository(nil)
	mailboxRepo := NewMemoryRepository()
	service := NewService(mailboxRepo, domainRepo)
	userID := uint64(7)

	_, err := domainRepo.Upsert(context.Background(), domain.Domain{
		Domain:            "example.com",
		OwnerUserID:       &userID,
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		VerificationScore: 100,
		HealthStatus:      "healthy",
		Weight:            100,
	})
	if err != nil {
		t.Fatalf("seed root domain: %v", err)
	}

	verifiedSubdomain, err := domainRepo.Upsert(context.Background(), domain.Domain{
		Domain:            "mx.example.com",
		OwnerUserID:       &userID,
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		VerificationScore: 100,
		HealthStatus:      "healthy",
		Weight:            90,
	})
	if err != nil {
		t.Fatalf("seed verified subdomain: %v", err)
	}

	_, err = domainRepo.Upsert(context.Background(), domain.Domain{
		Domain:            "relay.example.com",
		OwnerUserID:       &userID,
		Status:            "active",
		Visibility:        "private",
		PublicationStatus: "draft",
		VerificationScore: 0,
		HealthStatus:      "unknown",
		Weight:            90,
	})
	if err != nil {
		t.Fatalf("seed unverified subdomain: %v", err)
	}

	payload, err := service.BuildDashboard(context.Background(), userID)
	if err != nil {
		t.Fatalf("build dashboard: %v", err)
	}
	foundVerified := false
	foundRoot := false
	for _, item := range payload.AvailableDomains {
		if item.ID == verifiedSubdomain.ID {
			foundVerified = true
		}
		if item.Domain == "example.com" {
			foundRoot = true
		}
		if item.Domain == "relay.example.com" {
			t.Fatalf("expected unverified subdomain to be hidden from available domains")
		}
	}
	if !foundRoot {
		t.Fatalf("expected root domain to remain available")
	}
	if !foundVerified {
		t.Fatalf("expected verified subdomain to remain available")
	}
}

func pointerUint64(value uint64) *uint64 {
	return &value
}
