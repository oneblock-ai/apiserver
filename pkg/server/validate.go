package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/rancher/apiserver/pkg/apierror"
	"github.com/rancher/apiserver/pkg/parse"
	"github.com/rancher/apiserver/pkg/types"

	"github.com/rancher/wrangler/v3/pkg/schemas"
	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
)

const (
	csrfCookie = "CSRF"
	csrfHeader = "X-API-CSRF"
)

func ValidateAction(request *types.APIRequest) (*schemas.Action, error) {
	if request.Action == "" || request.Link != "" || request.Method != http.MethodPost {
		return nil, nil
	}

	if err := request.AccessControl.CanAction(request, request.Schema, request.Action); err != nil {
		return nil, err
	}

	actions := request.Schema.CollectionActions
	if request.Name != "" {
		actions = request.Schema.ResourceActions
	}

	action, ok := actions[request.Action]
	if !ok {
		return nil, apierror.NewAPIError(validation.InvalidAction, fmt.Sprintf("Invalid action: %s", request.Action))
	}

	return &action, nil
}

func CheckCSRF(apiOp *types.APIRequest) error {
	if !parse.IsBrowser(apiOp.RequestCtx.Request, false) {
		return nil
	}

	//cookie, err := apiOp.Request.Cookie(csrfCookie)
	cookie := apiOp.RequestCtx.Cookie(csrfCookie)
	//if errors.Is(err, http.ErrNoCookie) {
	if cookie == nil {
		// 16 bytes = 32 Hex Char = 128 bit entropy
		bytes := make([]byte, 16)
		_, err := rand.Read(bytes)
		if err != nil {
			return apierror.WrapAPIError(err, validation.ServerError, "Failed in CSRF processing")
		}

		//cookie = &http.Cookie{
		//	Name:   csrfCookie,
		//	Value:  hex.EncodeToString(bytes),
		//	Path:   "/",
		//	Secure: true,
		//}
		apiOp.RequestCtx.Request.SetCookie(csrfCookie, hex.EncodeToString(bytes))
		//http.SetCookie(apiOp.Response, cookie)
	}
	//} else if err != nil {
	//	return apierror.NewAPIError(validation.InvalidCSRFToken, "Failed to parse cookies")
	//} else if apiOp.Method != http.MethodGet {
	//	/*
	//	 * Very important to use apiOp.Method and not apiOp.Request.Method. The client can override the HTTP method with _method
	//	 */
	//	if cookie.Value == apiOp.Request.Header.Get(csrfHeader) {
	//		// Good
	//		return nil
	//	} else if cookie.Value == apiOp.Request.URL.Query().Get(csrfCookie) {
	//		// Good
	//		return nil
	//	} else {
	//		return apierror.NewAPIError(validation.InvalidCSRFToken, "Invalid CSRF token")
	//	}
	//}

	return nil
}
