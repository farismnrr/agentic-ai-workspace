package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
)

// serverMetadata is the document served for both RFC 8414 and OIDC Discovery.
type serverMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JWKSURI                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	ResponseModesSupported            []string `json:"response_modes_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ClaimsSupported                   []string `json:"claims_supported"`
	ClientIDMetadataDocumentSupported bool     `json:"client_id_metadata_document_supported"`
	AuthorizationResponseIssSupported bool     `json:"authorization_response_iss_parameter_supported"`
}

func newMetadata(issuer string, resources []application.Resource) serverMetadata {
	return serverMetadata{
		Issuer:                            issuer,
		AuthorizationEndpoint:             issuer + "/oauth/authorize",
		TokenEndpoint:                     issuer + "/oauth/token",
		RegistrationEndpoint:              issuer + "/oauth/register",
		RevocationEndpoint:                issuer + "/oauth/revoke",
		UserinfoEndpoint:                  issuer + "/oauth/userinfo",
		JWKSURI:                           issuer + "/oauth/jwks.json",
		ScopesSupported:                   supportedScopes(resources),
		ResponseTypesSupported:            []string{"code"},
		ResponseModesSupported:            []string{"query"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{"ES256"},
		ClaimsSupported:                   []string{"sub", "iss", "aud", "exp", "iat", "email", "email_verified", "name"},
		ClientIDMetadataDocumentSupported: true,
		AuthorizationResponseIssSupported: true,
	}
}

func supportedScopes(resources []application.Resource) []string {
	seen := map[string]bool{}
	scopes := []string{}

	add := func(scope domain.Scope) {
		if !seen[scope.ID] {
			seen[scope.ID] = true
			scopes = append(scopes, scope.ID)
		}
	}

	for _, scope := range domain.IdentityScopes {
		add(scope)
	}

	for _, resource := range resources {
		for _, scope := range resource.Scopes {
			add(scope)
		}
	}

	return scopes
}

func (h *Handler) serveMetadata(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "public, max-age=300")

	return c.JSON(h.metadata)
}

func (h *Handler) serveKeys(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "public, max-age=300")

	return c.JSON(h.keys.JWKS())
}
