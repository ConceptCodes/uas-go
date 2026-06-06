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
	certPEM := provider.ClientSecret.String()
	if certPEM == "" {
		return nil, errors.New("SAML provider requires a certificate in client_secret")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, errors.New("failed to decode SAML certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SAML certificate: %w", err)
	}

	privPEM := provider.ClientSecret.String()
	privBlock, _ := pem.Decode([]byte(privPEM))
	if privBlock == nil || privBlock.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("failed to decode SAML private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		key2, err2 := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse SAML private key: %w", err2)
		}
		rsaKey, ok := key2.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("SAML private key is not RSA")
		}
		return &tlsCertPair{PrivateKey: rsaKey, Certificate: cert}, nil
	}

	return &tlsCertPair{PrivateKey: key, Certificate: cert}, nil
}

func (h *SamlHandler) fetchIDPMetadata(metadataURL string) (*saml.EntityDescriptor, error) {
	resp, err := http.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IDP metadata: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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

	relayState := uuid.New().String()
	relayData := map[string]string{
		"provider_id":   providerID,
		"department_id": departmentID,
	}
	relayDataJSON, _ := json.Marshal(relayData)
	h.mfaHelper.StoreSSOState(relayState, map[string]string{"data": string(relayDataJSON)}, 10*time.Minute)

	redirectURL, err := sp.MakeRedirectAuthenticationRequest(relayState)
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

	providers, err := h.idPRepo.FindByDepartment("")
	if err != nil {
		h.responseHelper.SendErrorResponse(w, "No providers found", constants.NotFound, err)
		return
	}

	var provider *models.IdentityProvider
	for i, p := range providers {
		if p.ID == providerID {
			provider = &providers[i]
			break
		}
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

	providerUserID := assertion.Subject.NameID.Value
	departmentID := provider.DepartmentID

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
	if email != "" {
		user, _ = h.userRepo.FindByEmail(email, departmentID)
	}

	if user == nil {
		userID := uuid.New().String()
		user = &models.UserModel{
			ID:            userID,
			Email:         email,
			EmailVerified: email != "",
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
		h.deptRoleRepo.Create(&userRole)
	}

	identity := &models.UserIdentity{
		UserID:         user.ID,
		DepartmentID:   departmentID,
		ProviderID:     providerID,
		ProviderUserID: providerUserID,
		ProviderEmail:  email,
	}
	h.identityRepo.Create(identity)

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