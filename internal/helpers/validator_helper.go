package helpers

import (
	"fmt"
	"net/http"
	"strings"

	"uas/internal/constants"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

var validate *validator.Validate

type ValidatorHelper struct {
	log            zerolog.Logger
	responseHelper ResponseHelper
}

func init() {
	validate = validator.New()
	validate.RegisterValidation("noSQLKeywords", noSQLKeywords)
}

func NewValidatorHelper(log zerolog.Logger, responseHelper ResponseHelper) *ValidatorHelper {
	return &ValidatorHelper{log: log, responseHelper: responseHelper}
}

func noSQLKeywords(fl validator.FieldLevel) bool {
	sqlKeywords := []string{"SELECT", "FROM", "WHERE", "DELETE", "UPDATE", "INSERT", "DROP", "CREATE", "ALTER", "TRUNCATE"}

	value := fl.Field().String()
	for _, keyword := range sqlKeywords {
		if strings.Contains(strings.ToUpper(value), keyword) {
			return false
		}
	}
	return true
}

func (v *ValidatorHelper) ValidateStruct(w http.ResponseWriter, s interface{}) {
	v.log.Debug().Interface("struct", s).Msg("Validating Request Data")

	err := validate.Struct(s)
	if err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field validation for '%s' failed on the '%s' tag", err.Field(), err.Tag()))
		}
		v.responseHelper.SendErrorResponse(w, strings.Join(errMsgs, ", "), constants.BadRequest, err)
	}
}
