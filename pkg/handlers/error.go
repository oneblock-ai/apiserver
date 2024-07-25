package handlers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/types"

	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
	"github.com/sirupsen/logrus"
)

func ErrorHandler(request *types.APIRequest, err error) {
	if errors.Is(err, validation.ErrComplete) {
		return
	}

	var ec validation.ErrorCode
	if errors.As(err, &ec) {
		err = apierror.NewAPIError(ec, "")
	}

	var apiError *apierror.APIError
	if errors.As(err, &apiError) {
		if apiError.Cause != nil {
			//url, _ := url.PathUnescape(request.Request2.URL.String())
			url, _ := url.PathUnescape(request.RequestCtx.URI().String())
			if url == "" {
				//url = request.Request.URL.String()
				url = request.RequestCtx.Request.URI().String()
			}
			logrus.Errorf("API error response %v for %v %v. Cause: %v", apiError.Code.Status, request.RequestCtx.Method(),
				url, apiError.Cause)
		}
	}

	if apiError.Code.Status == http.StatusNoContent {
		request.RequestCtx.SetStatusCode(http.StatusNoContent)
		return
	}

	data := toError(apiError)
	request.WriteResponse(apiError.Code.Status, data)
}

func toError(apiError *apierror.APIError) types.APIObject {
	e := map[string]interface{}{
		"type":    "error",
		"status":  apiError.Code.Status,
		"code":    apiError.Code.Code,
		"message": apiError.Message,
	}
	if apiError.FieldName != "" {
		e["fieldName"] = apiError.FieldName
	}

	return types.APIObject{
		Type:   "error",
		Object: e,
	}
}
