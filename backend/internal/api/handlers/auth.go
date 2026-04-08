package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/auth"
	"github.com/stelgkio/affilflow/backend/internal/middleware"
	"github.com/stelgkio/affilflow/backend/pkg/response"
	"github.com/stelgkio/affilflow/backend/pkg/safeurl"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/google"
)

func (h *Handlers) oauthConfig(provider string) (*oauth2.Config, error) {
	base := strings.TrimRight(h.Cfg.AuthPublicBaseURL, "/")
	redirectURL := fmt.Sprintf("%s/api/v1/auth/providers/%s/callback", base, strings.ToLower(provider))
	switch strings.ToLower(provider) {
	case "google":
		if h.Cfg.OAuthGoogleClientID == "" || h.Cfg.OAuthGoogleClientSecret == "" {
			return nil, fmt.Errorf("google oauth not configured")
		}
		return &oauth2.Config{
			ClientID:     h.Cfg.OAuthGoogleClientID,
			ClientSecret: h.Cfg.OAuthGoogleClientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}, nil
	case "facebook":
		if h.Cfg.OAuthFacebookClientID == "" || h.Cfg.OAuthFacebookClientSecret == "" {
			return nil, fmt.Errorf("facebook oauth not configured")
		}
		return &oauth2.Config{
			ClientID:     h.Cfg.OAuthFacebookClientID,
			ClientSecret: h.Cfg.OAuthFacebookClientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"email", "public_profile"},
			Endpoint:     facebook.Endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider")
	}
}

// AuthOAuthStart GET /api/v1/auth/providers/:provider/start
func (h *Handlers) AuthOAuthStart(c *fiber.Ctx) error {
	provider := strings.ToLower(c.Params("provider"))
	cfg, err := h.oauthConfig(provider)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	next := c.Query("next", "")
	accountType := strings.ToLower(strings.TrimSpace(c.Query("account_type", "affiliate")))
	if accountType != "affiliate" && accountType != "merchant" {
		return fiber.NewError(fiber.StatusBadRequest, "account_type must be affiliate or merchant")
	}
	companyName := strings.TrimSpace(c.Query("company_name", ""))
	if accountType == "merchant" && companyName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "company_name is required for a company account")
	}
	state, err := auth.NewOAuthState(provider, next, accountType, companyName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "state")
	}
	u := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return c.Redirect(u, fiber.StatusFound)
}

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	VerifiedEmail bool   `json:"verified_email"`
}

type facebookUserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func fetchGoogleProfile(ctx context.Context, client *http.Client) (sub, email, name string, err error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("google userinfo: %s", string(b))
	}
	var u googleUserInfo
	if err := json.Unmarshal(b, &u); err != nil {
		return "", "", "", err
	}
	return u.ID, u.Email, u.Name, nil
}

func fetchFacebookProfile(ctx context.Context, client *http.Client) (sub, email, name string, err error) {
	resp, err := client.Get("https://graph.facebook.com/me?fields=id,name,email")
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", "", "", fmt.Errorf("facebook me: %s", string(b))
	}
	var u facebookUserInfo
	if err := json.Unmarshal(b, &u); err != nil {
		return "", "", "", err
	}
	return u.ID, u.Email, u.Name, nil
}

// AuthOAuthCallback GET /api/v1/auth/providers/:provider/callback
func (h *Handlers) AuthOAuthCallback(c *fiber.Ctx) error {
	provider := strings.ToLower(c.Params("provider"))
	if errMsg := c.Query("error"); errMsg != "" {
		return fiber.NewError(fiber.StatusBadRequest, c.Query("error_description", errMsg))
	}
	code := c.Query("code")
	state := c.Query("state")
	p, next, accountType, companyName, ok := auth.ConsumeOAuthState(state)
	if !ok || p != provider {
		return fiber.NewError(fiber.StatusBadRequest, "invalid oauth state")
	}

	oauthCfg, err := h.oauthConfig(provider)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx := c.UserContext()
	tok, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "token exchange failed")
	}
	client := oauthCfg.Client(ctx, tok)

	var provUID, email, displayName string
	switch provider {
	case "google":
		provUID, email, displayName, err = fetchGoogleProfile(ctx, client)
	case "facebook":
		provUID, email, displayName, err = fetchFacebookProfile(ctx, client)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unknown provider")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	if provUID == "" {
		return fiber.NewError(fiber.StatusBadGateway, "provider did not return id")
	}

	profileJSON, _ := json.Marshal(map[string]string{"name": displayName, "email": email})

	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}
	var namePtr *string
	if displayName != "" {
		namePtr = &displayName
	}

	userID, err := h.resolveOrCreateOAuthUser(ctx, provider, provUID, emailPtr, namePtr, profileJSON)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if err := h.applyMerchantOAuthSignup(ctx, userID, accountType, companyName); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	u, err := h.User.GetByID(ctx, userID)
	if err != nil || u == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "user load failed")
	}
	roles := []string{u.Role}
	if roles[0] == "" {
		roles = []string{"affiliate"}
	}

	jwtBytes, err := auth.IssueAccessToken(h.Cfg, userID, roles)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	tokenStr := string(jwtBytes)

	defaultNext := strings.TrimRight(h.Cfg.PublicAppBaseURL, "/") + "/auth/callback"
	redirectTo := safeurl.RedirectTarget(h.Cfg.RedirectAllowHosts, defaultNext, next)
	uu, err := url.Parse(redirectTo)
	if err != nil {
		redirectTo = defaultNext
		uu, _ = url.Parse(redirectTo)
	}
	q := uu.Query()
	q.Set("token", tokenStr)
	uu.RawQuery = q.Encode()
	return c.Redirect(uu.String(), fiber.StatusFound)
}

func (h *Handlers) resolveOrCreateOAuthUser(ctx context.Context, provider, providerUID string, email, displayName *string, profileJSON []byte) (string, error) {
	if uid, err := h.User.FindAuthIdentity(ctx, provider, providerUID); err != nil {
		return "", err
	} else if uid != "" {
		_ = h.User.UpsertAuthIdentity(ctx, provider, providerUID, uid, email, profileJSON)
		if displayName != nil || email != nil {
			_ = h.User.PatchEmailDisplayName(ctx, uid, email, displayName)
		}
		return uid, nil
	}

	if email != nil && *email != "" {
		if existing, err := h.User.GetByEmail(ctx, *email); err != nil {
			return "", err
		} else if existing != nil {
			if err := h.User.UpsertAuthIdentity(ctx, provider, providerUID, existing.ID, email, profileJSON); err != nil {
				return "", err
			}
			if displayName != nil || email != nil {
				_ = h.User.PatchEmailDisplayName(ctx, existing.ID, email, displayName)
			}
			return existing.ID, nil
		}
	}

	newID := uuid.NewString()
	role := "affiliate"
	if h.Cfg.AuthBootstrapAdminEmail != "" && email != nil &&
		strings.EqualFold(strings.TrimSpace(*email), strings.TrimSpace(h.Cfg.AuthBootstrapAdminEmail)) {
		role = "admin"
	}
	if err := h.User.CreateOAuthUser(ctx, newID, email, displayName, role); err != nil {
		return "", err
	}
	if err := h.User.UpsertAuthIdentity(ctx, provider, providerUID, newID, email, profileJSON); err != nil {
		return "", err
	}
	return newID, nil
}

func (h *Handlers) applyMerchantOAuthSignup(ctx context.Context, userID, accountType, companyName string) error {
	if strings.ToLower(accountType) != "merchant" {
		return nil
	}
	companyName = strings.TrimSpace(companyName)
	if companyName == "" {
		return nil
	}
	return h.ensureMerchantOrg(ctx, userID, companyName)
}

func (h *Handlers) ensureMerchantOrg(ctx context.Context, userID, companyName string) error {
	companyName = strings.TrimSpace(companyName)
	if companyName == "" {
		return fmt.Errorf("company name required")
	}
	u, err := h.User.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		return fmt.Errorf("user not found")
	}
	if u.OrganizationID != nil {
		return nil
	}
	orgID, err := h.Org.Create(ctx, companyName)
	if err != nil {
		return err
	}
	if h.Sub != nil {
		if err := h.Sub.CreateFree(ctx, orgID); err != nil {
			return err
		}
	}
	return h.User.SetOrganizationAndRole(ctx, userID, &orgID, "admin")
}

// AuthMe GET /api/v1/auth/me
func (h *Handlers) AuthMe(c *fiber.Ctx) error {
	uid, _ := c.Locals(middleware.LocalUserID).(string)
	roles, _ := c.Locals(middleware.LocalRoles).([]string)
	return response.JSON(c, 200, fiber.Map{"user_id": uid, "roles": roles})
}

// AuthLogout POST /api/v1/auth/logout — JWT is stateless; client discards token.
func (h *Handlers) AuthLogout(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}
