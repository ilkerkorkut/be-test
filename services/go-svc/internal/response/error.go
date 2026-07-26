package response

import "github.com/flotaro/be-test/services/go-svc/internal/enums"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorAttribute struct {
	AttributeName string `json:"attribute_name"`
	Message       string `json:"message"`
}

func NewErrorResponse(code string, message string, data any) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func NewValidationErrorResponse(errors map[string]string) ErrorResponse {
	if len(errors) == 0 {
		return ErrorResponse{
			Code:    enums.ValidationFailedCode,
			Message: enums.ValidationFailedMessage,
			Data:    nil,
		}
	}

	attrs := make([]ErrorAttribute, 0, len(errors))

	for k, v := range errors {
		attrs = append(attrs, ErrorAttribute{
			AttributeName: k,
			Message:       v,
		})
	}

	return ErrorResponse{
		Code:    enums.ValidationFailedCode,
		Message: enums.ValidationFailedMessage,
		Data:    attrs,
	}
}
