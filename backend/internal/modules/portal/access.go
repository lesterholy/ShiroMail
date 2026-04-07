package portal

import (
	"errors"
	"strings"
)

var ErrAPIKeyForbidden = errors.New("api key forbidden")

type BoundResource struct {
	ZoneID            *uint64
	NodeID            *uint64
	Visibility        string
	PublicationStatus string
	OwnerUserID       *uint64
}

func APIKeyHasScope(key APIKey, scope string) bool {
	needle := strings.TrimSpace(scope)
	if needle == "" {
		return false
	}

	for _, item := range key.Scopes {
		if strings.EqualFold(strings.TrimSpace(item), needle) {
			return true
		}
	}
	return false
}

func APIKeyHasRole(key APIKey, role string) bool {
	needle := strings.TrimSpace(role)
	if needle == "" {
		return false
	}

	for _, item := range key.Roles {
		if strings.EqualFold(strings.TrimSpace(item), needle) {
			return true
		}
	}
	return false
}

func APIKeyAllowsResourceClass(key APIKey, userID uint64, resource BoundResource) bool {
	mode := strings.TrimSpace(key.ResourcePolicy.DomainAccessMode)
	switch mode {
	case "public_only":
		if strings.TrimSpace(resource.Visibility) == "private" {
			return false
		}
	case "private_only":
		if strings.TrimSpace(resource.Visibility) != "private" {
			return false
		}
	}

	switch strings.TrimSpace(resource.Visibility) {
	case "private":
		if APIKeyHasRole(key, "admin") {
			return true
		}
		return key.ResourcePolicy.AllowOwnedPrivateDomains &&
			resource.OwnerUserID != nil &&
			*resource.OwnerUserID == userID
	case "platform_public":
		return key.ResourcePolicy.AllowPlatformPublicDomains &&
			isPublishedForSharedAccess(resource.PublicationStatus)
	case "public_pool":
		return key.ResourcePolicy.AllowUserPublishedDomains &&
			isPublishedForSharedAccess(resource.PublicationStatus)
	default:
		return false
	}
}

func APIKeyAllowsBoundResource(key APIKey, userID uint64, resource BoundResource, requiredAccess string) bool {
	if !APIKeyAllowsResourceClass(key, userID, resource) {
		return false
	}
	if !APIKeyHasDomainBindings(key) {
		return true
	}

	for _, binding := range key.DomainBindings {
		if !bindingAllowsAccess(binding.AccessLevel, requiredAccess) {
			continue
		}
		if bindingMatchesResource(binding, resource) {
			return true
		}
	}
	return false
}

func APIKeyAllowsExplicitPrivateBinding(key APIKey, resource BoundResource, requiredAccess string) bool {
	if strings.TrimSpace(resource.Visibility) != "private" {
		return false
	}
	if !key.ResourcePolicy.AllowOwnedPrivateDomains {
		return false
	}
	if strings.TrimSpace(key.ResourcePolicy.DomainAccessMode) == "public_only" {
		return false
	}
	if !APIKeyHasDomainBindings(key) {
		return false
	}

	for _, binding := range key.DomainBindings {
		if !bindingAllowsAccess(binding.AccessLevel, requiredAccess) {
			continue
		}
		if bindingMatchesResource(binding, resource) {
			return true
		}
	}
	return false
}

func APIKeyHasDomainBindings(key APIKey) bool {
	return len(key.DomainBindings) > 0
}

func apiKeyPrefix(preview string) string {
	trimmed := strings.TrimSpace(preview)
	if len(trimmed) <= 10 {
		return trimmed
	}
	return trimmed[:10]
}

func isPublishedForSharedAccess(status string) bool {
	switch strings.TrimSpace(status) {
	case "", "published", "approved", "active":
		return true
	default:
		return false
	}
}

func bindingAllowsAccess(binding string, required string) bool {
	current := strings.TrimSpace(binding)
	need := strings.TrimSpace(required)
	if current == "" || need == "" {
		return false
	}
	if current == "manage" {
		return true
	}
	if current == need {
		return true
	}

	switch current {
	case "write":
		return need == "read"
	case "verify":
		return need == "read"
	case "publish":
		return need == "read"
	default:
		return false
	}
}

func bindingMatchesResource(binding APIKeyDomainBinding, resource BoundResource) bool {
	if binding.ZoneID == nil && binding.NodeID == nil {
		return false
	}
	if binding.ZoneID != nil {
		if resource.ZoneID == nil || *binding.ZoneID != *resource.ZoneID {
			return false
		}
	}
	if binding.NodeID != nil {
		if resource.NodeID == nil || *binding.NodeID != *resource.NodeID {
			return false
		}
	}
	return true
}
