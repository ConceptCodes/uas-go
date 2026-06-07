package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type SsoHandler struct {
	idPRepo        repository.IdentityProviderRepository
	identityRepo   repository.UserIdentityRepository
	userRepo       repository.UserRepository
	deptRoleRepo   repository.DepartmentRoleRepository
	sessionRepo    repository.SessionRepository
	authHelper     *helpers.AuthHelper
	ssoHelper      *helpers.SSOHelper
	mfaHelper      *helpers.MfaHelper
	responseHelper *helpers.ResponseHelper
	validatorHelper *helpers.ValidatorHelper
	log            *zerolog.Logger
}

func NewSsoHandler(
	idPRepo repository.IdentityProviderRepository,
	identityRepo repository.UserIdentityRepository,
	userRepo repository.UserRepository,
	deptRoleRepo repository.DepartmentRoleRepository,
	sessionRepo repository.SessionRepository,
	authHelper *helpers.AuthHelper,
	ssoHelper *helpers.SSOHelper,
	mfaHelper *helpers.MfaHelper,
	responseHelper *helpers.ResponseHelper,
	validatorHelper *helpers.ValidatorHelper,
	log *zerolog.Logger,
) *SsoHandler {
	return &SsoHandler{
		idPRepo:        idPRepo,
		identityRepo:   identityRepo,
		userRepo:       userRepo,
		deptRoleRepo:   deptRoleRepo,
		sessionRepo:    sessionRepo,
		authHelper:     authHelper,
		ssoHelper:      ssoHelper,
		mfaHelper:      mfaHelper,
		responseHelper: responseHelper,
		validatorHelper: validatorHelper,
		log:            log,
	}
}

func (h *SsoHandler) CreateProviderHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	role := helpers.GetRole(r)

	if role != models.Admin {
		h.responseHelper.SendErrorResponse(w, "Admin access required", constants.Forbidden, nil)
		return
	}

	var data models.CreateIdPRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	redirectURLs := models.SerializeStringSlice(data.RedirectURLs)
	scopes := models.SerializeStringSlice(data.Scopes)

	provider := &models.IdentityProvider{
		DepartmentID:     departmentID,
		Name:             data.Name,
		ProviderType:     data.ProviderType,
		ClientID:         data.ClientID,
		IssuerURL:        data.IssuerURL,
		AuthorizationURL: data.AuthorizationURL,
		TokenURL:         data.TokenURL,
		UserInfoURL:      data.UserInfoURL,
		JWKSURI:          data.JWKSURI,
		MetadataURL:      data.MetadataURL,
		RedirectURLs:     redirectURLs,
		Scopes:           scopes,
		Enabled:          true,
	}
	provider.ClientSecret.Set(data.ClientSecret)

	if err := h.idPRepo.Create(provider); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to create identity provider", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Identity provider created", h.toProviderResponse(provider))
}

func (h *SsoHandler) ListProvidersHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)

	providers, err := h.idPRepo.FindByDepartment(departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to list providers", constants.InternalServerError, err)
		return
	}

	var response []models.IdentityProviderResponse
	for _, p := range providers {
		response = append(response, h.toProviderResponse(&p))
	}

	h.responseHelper.SendSuccessResponse(w, "Providers retrieved", response)
}

func (h *SsoHandler) GetProviderHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	provider, err := h.idPRepo.FindByID(id, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Provider not found", constants.NotFound, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Provider retrieved", h.toProviderResponse(provider))
}

func (h *SsoHandler) UpdateProviderHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	role := helpers.GetRole(r)
	id := mux.Vars(r)["id"]

	if role != models.Admin {
		h.responseHelper.SendErrorResponse(w, "Admin access required", constants.Forbidden, nil)
		return
	}

	provider, err := h.idPRepo.FindByID(id, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Provider not found", constants.NotFound, err)
		return
	}

	var data models.UpdateIdPRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if data.Name != nil {
		provider.Name = *data.Name
	}
	if data.ClientID != nil {
		provider.ClientID = *data.ClientID
	}
	if data.ClientSecret != nil {
		provider.ClientSecret.Set(*data.ClientSecret)
	}
	if data.IssuerURL != nil {
		provider.IssuerURL = *data.IssuerURL
	}
	if data.AuthorizationURL != nil {
		provider.AuthorizationURL = *data.AuthorizationURL
	}
	if data.TokenURL != nil {
		provider.TokenURL = *data.TokenURL
	}
	if data.UserInfoURL != nil {
		provider.UserInfoURL = *data.UserInfoURL
	}
	if data.JWKSURI != nil {
		provider.JWKSURI = *data.JWKSURI
	}
	if data.MetadataURL != nil {
		provider.MetadataURL = *data.MetadataURL
	}
	if data.RedirectURLs != nil {
		provider.RedirectURLs = models.SerializeStringSlice(data.RedirectURLs)
	}
	if data.Scopes != nil {
		provider.Scopes = models.SerializeStringSlice(data.Scopes)
	}
	if data.Enabled != nil {
		provider.Enabled = *data.Enabled
	}

	if err := h.idPRepo.Update(provider); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to update provider", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Provider updated", h.toProviderResponse(provider))
}

func (h *SsoHandler) DeleteProviderHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)
	role := helpers.GetRole(r)
	id := mux.Vars(r)["id"]

	if role != models.Admin {
		h.responseHelper.SendErrorResponse(w, "Admin access required", constants.Forbidden, nil)
		return
	}

	if err := h.idPRepo.Delete(id, departmentID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to delete provider", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Provider deleted", nil)
}

func (h *SsoHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	departmentID := helpers.GetDepartmentId(r)

	var data models.SsoLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	if !h.validatorHelper.ValidateStruct(w, &data) {
		return
	}

	provider, err := h.idPRepo.FindByID(data.ProviderID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Provider not found", constants.NotFound, err)
		return
	}

	if !provider.Enabled {
		h.responseHelper.SendErrorResponse(w, "Provider is disabled", constants.Forbidden, nil)
		return
	}

	state, err := h.ssoHelper.GenerateRandomState()
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate state", constants.InternalServerError, err)
		return
	}

	codeVerifier, codeChallenge, err := h.mfaHelper.GenerateCodeVerifier()
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate PKCE", constants.InternalServerError, err)
		return
	}

	stateData := map[string]string{
		"provider_id":   data.ProviderID,
		"department_id": departmentID,
		"redirect_uri":  data.RedirectURI,
		"code_verifier": codeVerifier,
	}
	stateTTL := time.Duration(config.AppConfig.SsoStateExpireMin) * time.Minute
	if err := h.mfaHelper.StoreSSOState(state, stateData, stateTTL); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to store state", constants.InternalServerError, err)
		return
	}

	authURL := h.ssoHelper.BuildAuthorizationURL(provider, state, codeChallenge, data.RedirectURI)

	h.responseHelper.SendSuccessResponse(w, "SSO login initiated", models.SsoLoginResponse{
		AuthURL: authURL,
		State:   state,
	})
}

func (h *SsoHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	var data models.SsoCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.responseHelper.SendErrorResponse(w, err.Error(), constants.BadRequest, err)
		return
	}

	stateData, err := h.mfaHelper.ValidateAndConsumeSSOState(data.State)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Invalid or expired state", constants.Unauthorized, err)
		return
	}

	departmentID := stateData["department_id"]
	providerID := stateData["provider_id"]
	redirectURI := stateData["redirect_uri"]
	codeVerifier := stateData["code_verifier"]

	provider, err := h.idPRepo.FindByID(providerID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Provider not found", constants.NotFound, err)
		return
	}

	var tokenResult map[string]interface{}
	var providerUserID, email string
	emailVerified := false

	if provider.ProviderType == models.IDPGithub {
		tokenResult, err = h.ssoHelper.GenerateGitHubToken(data.Code, provider.ClientID, provider.ClientSecret.String(), redirectURI)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Token exchange failed", constants.InternalServerError, err)
			return
		}
		accessToken, _ := tokenResult["access_token"].(string)
		var userInfo map[string]interface{}
		userInfo, err = h.ssoHelper.GetUserInfo(provider, accessToken)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Failed to get user info", constants.InternalServerError, err)
			return
		}
		providerUserID, email, err = h.ssoHelper.ExtractIdentity(userInfo, provider)
		emailVerified = true
	} else {
		tokenResult, err = h.ssoHelper.ExchangeCodeForToken(provider, data.Code, codeVerifier, redirectURI)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "Token exchange failed", constants.InternalServerError, err)
			return
		}

		if idToken, ok := tokenResult["id_token"].(string); ok && idToken != "" {
			claims, _ := h.ssoHelper.ParseIDTokenUnverified(idToken)
			if sub, ok := claims["sub"].(string); ok {
				providerUserID = sub
			}
			if em, ok := claims["email"].(string); ok {
				email = em
			}
			if ev, ok := claims["email_verified"].(bool); ok {
				emailVerified = ev
			}
		}

		if providerUserID == "" {
			accessToken, _ := tokenResult["access_token"].(string)
			var userInfo map[string]interface{}
			userInfo, err = h.ssoHelper.GetUserInfo(provider, accessToken)
			if err != nil {
				h.responseHelper.SendErrorResponse(w, "Failed to get user info", constants.InternalServerError, err)
				return
			}
			providerUserID, email, err = h.ssoHelper.ExtractIdentity(userInfo, provider)
		}
	}

	if err != nil || providerUserID == "" {
		h.responseHelper.SendErrorResponse(w, "Failed to extract user identity", constants.InternalServerError, err)
		return
	}

	existingIdentity, err := h.identityRepo.FindByProvider(providerID, providerUserID, departmentID)
	if err == nil && existingIdentity != nil {
		user, err := h.userRepo.FindById(existingIdentity.UserID, departmentID)
		if err != nil {
			h.responseHelper.SendErrorResponse(w, "User not found", constants.InternalServerError, err)
			return
		}
		h.identityRepo.UpdateLastLogin(existingIdentity.ID, departmentID)
		h.issueTokens(w, r, user, departmentID)
		return
	}

	var user *models.UserModel
	if email != "" && emailVerified {
		user, _ = h.userRepo.FindByEmail(email, departmentID)
	}

	if user == nil {
		userID := uuid.New().String()
		user = &models.UserModel{
			ID:            userID,
			Email:         email,
			EmailVerified: emailVerified && email != "",
		}
		if err := h.userRepo.Create(user); err != nil {
			h.responseHelper.SendErrorResponse(w, "Failed to create user", constants.InternalServerError, err)
			return
		}
		userRole := models.DepartmentRoles{
			ID:     departmentID,
			Role:   models.User,
			UserID: userID,
		}
		if err := h.deptRoleRepo.Create(&userRole); err != nil {
			h.log.Error().Err(err).Str("user_id", userID).Str("department_id", departmentID).Msg("Failed to assign user role")
		}
	}

	identity := &models.UserIdentity{
		UserID:         user.ID,
		DepartmentID:   departmentID,
		ProviderID:     providerID,
		ProviderUserID: providerUserID,
		ProviderEmail:  email,
	}
	if accessToken, ok := tokenResult["access_token"].(string); ok {
		identity.AccessToken = accessToken
	}
	if refreshToken, ok := tokenResult["refresh_token"].(string); ok {
		identity.RefreshToken = refreshToken
	}
	if err := h.identityRepo.Create(identity); err != nil {
		h.log.Error().Err(err).Str("user_id", user.ID).Msg("Failed to link SSO identity")
	}

	h.issueTokens(w, r, user, departmentID)
}

func (h *SsoHandler) issueTokens(w http.ResponseWriter, r *http.Request, user *models.UserModel, departmentID string) {
	accessToken, err := h.authHelper.GenerateAccessJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate access token", constants.InternalServerError, err)
		return
	}

	refreshToken, err := h.authHelper.GenerateRefreshJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate refresh token", constants.InternalServerError, err)
		return
	}

	h.authHelper.GenerateAccessCookie(accessToken, w)
	w.Header().Set(constants.JwtHeader, refreshToken)

	h.responseHelper.SendSuccessResponse(w, "SSO login successful", nil)
}

func (h *SsoHandler) ListIdentitiesHandler(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserId(r)
	departmentID := helpers.GetDepartmentId(r)

	if userID == "" {
		h.responseHelper.SendErrorResponse(w, "Authentication required", constants.Unauthorized, nil)
		return
	}

	identities, err := h.identityRepo.FindByUserID(userID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to list identities", constants.InternalServerError, err)
		return
	}

	var response []models.UserIdentityResponse
	for _, id := range identities {
		provider, _ := h.idPRepo.FindByID(id.ProviderID, departmentID)
		providerName := ""
		providerType := ""
		if provider != nil {
			providerName = provider.Name
			providerType = string(provider.ProviderType)
		}
		response = append(response, models.UserIdentityResponse{
			ID:             id.ID,
			ProviderID:     id.ProviderID,
			ProviderName:   providerName,
			ProviderType:   providerType,
			ProviderUserID: id.ProviderUserID,
			ProviderEmail:  id.ProviderEmail,
			LastLoginAt:    id.LastLoginAt,
			CreatedAt:      id.CreatedAt,
		})
	}

	h.responseHelper.SendSuccessResponse(w, "Identities retrieved", response)
}

func (h *SsoHandler) DeleteIdentityHandler(w http.ResponseWriter, r *http.Request) {
	userID := helpers.GetUserId(r)
	departmentID := helpers.GetDepartmentId(r)
	id := mux.Vars(r)["id"]

	if userID == "" {
		h.responseHelper.SendErrorResponse(w, "Authentication required", constants.Unauthorized, nil)
		return
	}

	identity, err := h.identityRepo.FindByID(id, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Identity not found", constants.NotFound, err)
		return
	}

	if identity.UserID != userID {
		h.responseHelper.SendErrorResponse(w, "Identity does not belong to user", constants.Forbidden, nil)
		return
	}

	if err := h.identityRepo.Delete(id, departmentID); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to delete identity", constants.InternalServerError, err)
		return
	}

	h.responseHelper.SendSuccessResponse(w, "Identity unlinked", nil)
}

func (h *SsoHandler) toProviderResponse(p *models.IdentityProvider) models.IdentityProviderResponse {
	return models.IdentityProviderResponse{
		ID:              p.ID,
		Name:            p.Name,
		ProviderType:    string(p.ProviderType),
		ClientID:        p.ClientID,
		IssuerURL:       p.IssuerURL,
		AuthorizationURL: p.AuthorizationURL,
		RedirectURLs:    models.DeserializeStringSlice(p.RedirectURLs),
		Scopes:          models.DeserializeStringSlice(p.Scopes),
		Enabled:         p.Enabled,
		CreatedAt:       p.CreatedAt,
	}
}