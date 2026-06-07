package handlers

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"uas/config"
	"uas/internal/constants"
	"uas/internal/helpers"
	"uas/internal/models"
	repository "uas/internal/repositories"

	"github.com/crewjam/saml"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

type SamlHandler struct {
	idPRepo        repository.IdentityProviderRepository
	identityRepo   repository.UserIdentityRepository
	userRepo       repository.UserRepository
	deptRoleRepo   repository.DepartmentRoleRepository
	sessionRepo    repository.SessionRepository
	authHelper     *helpers.AuthHelper
	mfaHelper      *helpers.MfaHelper
	responseHelper *helpers.ResponseHelper
	log            *zerolog.Logger
	samlSPs        map[string]*saml.ServiceProvider
}

func NewSamlHandler(
	idPRepo repository.IdentityProviderRepository,
	identityRepo repository.UserIdentityRepository,
	userRepo repository.UserRepository,
	deptRoleRepo repository.DepartmentRoleRepository,
	sessionRepo repository.SessionRepository,
	authHelper *helpers.AuthHelper,
	mfaHelper *helpers.MfaHelper,
	responseHelper *helpers.ResponseHelper,
	log *zerolog.Logger,
) *SamlHandler {
	return &SamlHandler{
		idPRepo:        idPRepo,
		identityRepo:   identityRepo,
		userRepo:       userRepo,
		deptRoleRepo:   deptRoleRepo,
		sessionRepo:    sessionRepo,
		authHelper:     authHelper,
		mfaHelper:      mfaHelper,
		responseHelper: responseHelper,
		log:            log,
		samlSPs:        make(map[string]*saml.ServiceProvider),
	}
}

func (h *SamlHandler) getOrCreateSP(provider *models.IdentityProvider) (*saml.ServiceProvider, error) {
	if sp, ok := h.samlSPs[provider.ID]; ok {
		return sp, nil
	}

	keyPair, err := h.getTLSCert(provider)
	if err != nil {
		return nil, err
	}

	rootURL := fmt.Sprintf("http://localhost:%d", config.AppConfig.Port)
	if config.AppConfig.Env == "production" {
		rootURL = fmt.Sprintf("https://%s", config.AppConfig.Host)
	}

	metadataURL, _ := url.Parse(rootURL + constants.ApiPrefix + "/saml/" + provider.ID + "/metadata")
	acsURL, _ := url.Parse(rootURL + constants.ApiPrefix + "/saml/" + provider.ID + "/acs")

	var redirectURLs []string
	if provider.RedirectURLs != "" {
		redirectURLs = models.DeserializeStringSlice(provider.RedirectURLs)
		if len(redirectURLs) > 0 {
			acsURL, _ = url.Parse(redirectURLs[0])
		}
	}

	sp := &saml.ServiceProvider{
		Key:         keyPair.PrivateKey,
		Certificate: keyPair.Certificate,
		MetadataURL: *metadataURL,
		AcsURL:      *acsURL,
	}

	if provider.IssuerURL != "" {
		sp.EntityID = provider.IssuerURL
	}

	if provider.MetadataURL != "" {
		idpMetadata, err := h.fetchIDPMetadata(provider.MetadataURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch IDP metadata: %w", err)
		}
		sp.IDPMetadata = idpMetadata
	}

	h.samlSPs[provider.ID] = sp
	return sp, nil
}

type tlsCertPair struct {
	PrivateKey  *rsa.PrivateKey
	Certificate *x509.Certificate
}

func (h *SamlHandler) getTLSCert(provider *models.IdentityProvider) (*tlsCertPair, error) {
	combinedPEM := provider.ClientSecret.String()
	if combinedPEM == "" {
		return nil, errors.New("SAML provider requires a certificate and private key in client_secret")
	}

	remaining := []byte(combinedPEM)
	var cert *x509.Certificate
	for {
		var block *pem.Block
		block, remaining = pem.Decode(remaining)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" && cert == nil {
			c, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SAML certificate: %w", err)
			}
			cert = c
		}
	}

	if cert == nil {
		return nil, errors.New("SAML provider PEM is missing a CERTIFICATE block")
	}

	keyPEM := []byte(combinedPEM)
	var privKey *rsa.PrivateKey
	for {
		var block *pem.Block
		block, keyPEM = pem.Decode(keyPEM)
		if block == nil {
			break
		}
		if block.Type == "RSA PRIVATE KEY" {
			k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
			}
			privKey = k
			break
		}
		if block.Type == "PRIVATE KEY" {
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
			}
			rk, ok := k.(*rsa.PrivateKey)
			if !ok {
				return nil, errors.New("SAML private key is not RSA")
			}
			privKey = rk
			break
		}
	}

	if privKey == nil {
		return nil, errors.New("SAML provider PEM is missing an RSA private key")
	}

	return &tlsCertPair{PrivateKey: privKey, Certificate: cert}, nil
}

func (h *SamlHandler) fetchIDPMetadata(metadataURL string) (*saml.EntityDescriptor, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IDP metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IDP metadata fetch returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read IDP metadata: %w", err)
	}

	var entityDescriptor saml.EntityDescriptor
	if err := xml.Unmarshal(body, &entityDescriptor); err != nil {
		return nil, fmt.Errorf("failed to parse IDP metadata: %w", err)
	}

	return &entityDescriptor, nil
}

func (h *SamlHandler) GetMetadataHandler(w http.ResponseWriter, r *http.Request) {
	providerID := mux.Vars(r)["id"]
	departmentID := helpers.GetDepartmentId(r)

	provider, err := h.idPRepo.FindByID(providerID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Provider not found", constants.NotFound, err)
		return
	}

	sp, err := h.getOrCreateSP(provider)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to create SP", constants.InternalServerError, err)
		return
	}

	metadata := sp.Metadata()

	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	if err := xml.NewEncoder(w).Encode(metadata); err != nil {
		h.log.Error().Err(err).Msg("Failed to encode SAML metadata")
	}
}

func (h *SamlHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	providerID := mux.Vars(r)["id"]
	departmentID := helpers.GetDepartmentId(r)

	provider, err := h.idPRepo.FindByID(providerID, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Provider not found", constants.NotFound, err)
		return
	}

	if !provider.Enabled {
		h.responseHelper.SendErrorResponse(w, "Provider is disabled", constants.Forbidden, nil)
		return
	}

	sp, err := h.getOrCreateSP(provider)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to create SP", constants.InternalServerError, err)
		return
	}

	authnReq, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding), saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to build authentication request", constants.InternalServerError, err)
		return
	}

	relayState := uuid.New().String()
	relayData := map[string]string{
		"provider_id":     providerID,
		"department_id":   departmentID,
		"saml_request_id": string(authnReq.ID),
	}
	relayDataJSON, _ := json.Marshal(relayData)
	if err := h.mfaHelper.StoreSSOState(relayState, map[string]string{"data": string(relayDataJSON)}, 10*time.Minute); err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to store SAML state", constants.InternalServerError, err)
		return
	}

	redirectURL, err := authnReq.Redirect(relayState, sp)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to build authentication request", constants.InternalServerError, err)
		return
	}

	if r.URL.Query().Get("format") == "json" {
		h.responseHelper.SendSuccessResponse(w, "SAML login initiated", map[string]string{
			"redirectUrl": redirectURL.String(),
		})
		return
	}

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (h *SamlHandler) AssertionConsumerServiceHandler(w http.ResponseWriter, r *http.Request) {
	providerID := mux.Vars(r)["id"]

	relayState := r.FormValue("RelayState")
	if relayState == "" {
		relayState = r.URL.Query().Get("RelayState")
	}

	var provider *models.IdentityProvider
	var departmentID string
	var samlRequestID string

	if relayState != "" {
		stateMap, err := h.mfaHelper.ValidateAndConsumeSSOState(relayState)
		if err == nil && stateMap != nil {
			var relayData map[string]string
			if dataStr, ok := stateMap["data"]; ok {
				_ = json.Unmarshal([]byte(dataStr), &relayData)
			}
			if id, ok := relayData["provider_id"]; ok && id == providerID {
				departmentID = relayData["department_id"]
				samlRequestID = relayData["saml_request_id"]
			}
		}
	}

	if departmentID == "" {
		departmentID = helpers.GetDepartmentId(r)
	}

	var err error
	if departmentID != "" {
		provider, err = h.idPRepo.FindByID(providerID, departmentID)
	}

	if provider == nil {
		h.responseHelper.SendErrorResponse(w, "Provider not found", constants.NotFound, nil)
		return
	}

	sp, err := h.getOrCreateSP(provider)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to create SP", constants.InternalServerError, err)
		return
	}

	var possibleRequestIDs []string
	if samlRequestID != "" {
		possibleRequestIDs = append(possibleRequestIDs, samlRequestID)
	}
	assertion, err := sp.ParseResponse(r, possibleRequestIDs)
	if err != nil {
		if relayStateErr := new(saml.InvalidResponseError); errors.As(err, &relayStateErr) {
			h.responseHelper.SendErrorResponse(w, "Invalid SAML response", constants.Unauthorized, err)
			return
		}
		h.responseHelper.SendErrorResponse(w, "Failed to parse SAML response", constants.Unauthorized, err)
		return
	}

	userAttributes := make(map[string]string)
	for _, attrStatement := range assertion.AttributeStatements {
		for _, attr := range attrStatement.Attributes {
			if len(attr.Values) > 0 {
				userAttributes[attr.Name] = attr.Values[0].Value
			}
		}
	}

	email := userAttributes["email"]
	if email == "" {
		email = userAttributes["Email"]
	}
	if email == "" && assertion.Subject.NameID != nil {
		email = assertion.Subject.NameID.Value
	}

	if assertion.Subject == nil || assertion.Subject.NameID == nil || assertion.Subject.NameID.Value == "" {
		h.responseHelper.SendErrorResponse(w, "SAML assertion missing subject", constants.BadRequest, nil)
		return
	}
	providerUserID := assertion.Subject.NameID.Value
	departmentID = provider.DepartmentID

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

	userID := uuid.New().String()
	user := &models.UserModel{
		ID:            userID,
		Email:         email,
		EmailVerified: false,
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

	identity := &models.UserIdentity{
		UserID:         user.ID,
		DepartmentID:   departmentID,
		ProviderID:     providerID,
		ProviderUserID: providerUserID,
		ProviderEmail:  email,
	}
	if err := h.identityRepo.Create(identity); err != nil {
		h.log.Error().Err(err).Str("user_id", user.ID).Msg("Failed to link SAML identity")
	}

	h.issueTokens(w, r, user, departmentID)
}

func (h *SamlHandler) issueTokens(w http.ResponseWriter, r *http.Request, user *models.UserModel, departmentID string) {
	accessToken, err := h.authHelper.GenerateAccessJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate tokens", constants.InternalServerError, err)
		return
	}

	refreshToken, err := h.authHelper.GenerateRefreshJwtToken(user, departmentID)
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "Failed to generate tokens", constants.InternalServerError, err)
		return
	}

	h.authHelper.GenerateAccessCookie(accessToken, w)
	w.Header().Set(constants.JwtHeader, refreshToken)
	h.responseHelper.SendSuccessResponse(w, "SAML login successful", nil)
}