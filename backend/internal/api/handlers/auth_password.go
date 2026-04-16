package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stelgkio/affilflow/backend/internal/auth"
	"github.com/stelgkio/affilflow/backend/pkg/response"
	"golang.org/x/crypto/bcrypt"
)

type registerBody struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	AccountType string `json:"account_type"` // affiliate | merchant
	CompanyName string `json:"company_name"`
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthRegister POST /api/v1/auth/register — email/password sign-up.
func (h *Handlers) AuthRegister(c *fiber.Ctx) error {
	var body registerBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	email := strings.TrimSpace(body.Email)
	pw := body.Password
	if email == "" || pw == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}
	if len(pw) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "password must be at least 8 characters")
	}
	accountType := strings.ToLower(strings.TrimSpace(body.AccountType))
	if accountType == "" {
		accountType = "affiliate"
	}
	if accountType != "affiliate" && accountType != "merchant" {
		return fiber.NewError(fiber.StatusBadRequest, "account_type must be affiliate or merchant")
	}
	companyName := strings.TrimSpace(body.CompanyName)
	if accountType == "merchant" && companyName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "company_name is required for a company account")
	}

	ctx := c.UserContext()
	if existing, err := h.User.GetByEmail(ctx, email); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	} else if existing != nil {
		return fiber.NewError(fiber.StatusConflict, "email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "password hash failed")
	}
	uid := uuid.NewString()
	var emailPtr *string = &email
	var dispPtr *string
	if strings.TrimSpace(body.DisplayName) != "" {
		s := strings.TrimSpace(body.DisplayName)
		dispPtr = &s
	}
	hashStr := string(hash)

	if accountType == "affiliate" {
		tx, err := h.Pool.Begin(ctx)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := h.User.InsertPasswordUserTx(ctx, tx, uid, emailPtr, dispPtr, hashStr, "affiliate", nil); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if err := tx.Commit(ctx); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return h.issueAuthResponse(c, fiber.StatusCreated, uid, []string{"affiliate"}, nil)
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()

	campainID, err := h.Campain.CreateTx(ctx, tx, companyName, &uid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if h.Sub != nil {
		if err := h.Sub.CreateFreeTx(ctx, tx, campainID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	if err := h.User.InsertPasswordUserTx(ctx, tx, uid, emailPtr, dispPtr, hashStr, "merchant", &campainID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	campainStr := campainID.String()
	return h.issueAuthResponse(c, fiber.StatusCreated, uid, []string{"merchant"}, &campainStr)
}

// AuthLogin POST /api/v1/auth/login — email/password; returns JWT.
func (h *Handlers) AuthLogin(c *fiber.Ctx) error {
	var body loginBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json")
	}
	email := strings.TrimSpace(body.Email)
	pw := body.Password
	if email == "" || pw == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}
	ctx := c.UserContext()
	u, err := h.User.GetByEmail(ctx, email)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if u == nil || u.PasswordHash == nil || *u.PasswordHash == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(pw)); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}
	role := u.Role
	if role == "" {
		role = "affiliate"
	}
	return h.issueAuthResponse(c, fiber.StatusOK, u.ID, []string{role}, nil)
}

func (h *Handlers) issueAuthResponse(c *fiber.Ctx, status int, userID string, roles []string, campainID *string) error {
	raw, err := auth.IssueAccessToken(h.Cfg, userID, roles)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	out := fiber.Map{
		"access_token": string(raw),
		"user_id":      userID,
		"roles":        roles,
	}
	if campainID != nil {
		out["campain_id"] = *campainID
	}
	return response.JSON(c, status, out)
}
