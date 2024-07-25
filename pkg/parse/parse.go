package parse

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"

	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/apiserver/pkg/urlbuilder"
)

const (
	maxFormSize = 2 * 1 << 20
)

var (
	allowedFormats = map[string]bool{
		"html":  true,
		"json":  true,
		"jsonl": true,
		"yaml":  true,
	}
)

type ParsedURL struct {
	Type       string
	Name       string
	Namespace  string
	Link       string
	Method     string
	Action     string
	Prefix     string
	SubContext map[string]string
	Query      url.Values
	QueryArgs  *protocol.Args
}

type URLParser func(rw http.ResponseWriter, req *http.Request, schemas *types.APISchemas) (ParsedURL, error)
type URLParser2 func(c *app.RequestContext, schemas *types.APISchemas) (ParsedURL, error)

type Parser func(apiOp *types.APIRequest, urlParser URLParser2) error

func Parse(apiOp *types.APIRequest, urlParser URLParser2) error {
	var err error

	//if apiOp.Request == nil {
	//	apiOp.Request, err = http.NewRequest("GET", "/", nil)
	//	if err != nil {
	//		return err
	//	}
	//}
	//
	if apiOp.RequestCtx == nil {
		return fmt.Errorf("request context can't be empty")
	}

	apiOp = types.StoreAPIContext(apiOp)

	if len(apiOp.Method) == 0 {
		//apiOp.Method = parseMethod(apiOp.Request)
		apiOp.Method = string(apiOp.RequestCtx.Method())
	}
	if apiOp.ResponseFormat == "" {
		//apiOp.RequestCtx.ContentType()
		apiOp.ResponseFormat = parseResponseFormat(apiOp.RequestCtx)
	}

	// The response format is guaranteed to be set even in the event of an error
	//parsedURL, err := urlParser(apiOp.Response, apiOp.Request, apiOp.Schemas)
	parsedURL, err := urlParser(apiOp.RequestCtx, apiOp.Schemas)
	// wait to check error, want to set as much as possible

	if apiOp.Type == "" {
		apiOp.Type = parsedURL.Type
	}
	if apiOp.Name == "" {
		apiOp.Name = parsedURL.Name
	}
	if apiOp.Link == "" {
		apiOp.Link = parsedURL.Link
	}
	if apiOp.Action == "" {
		apiOp.Action = parsedURL.Action
	}
	if apiOp.Query == nil {
		apiOp.Query = parsedURL.Query
	}
	if apiOp.Method == "" && parsedURL.Method != "" {
		apiOp.Method = parsedURL.Method
	}
	if apiOp.URLPrefix == "" {
		apiOp.URLPrefix = parsedURL.Prefix
	}
	if apiOp.Namespace == "" {
		apiOp.Namespace = parsedURL.Namespace
	}

	if apiOp.URLBuilder == nil {
		// make error local to not override the outer error we have yet to check
		var err error
		apiOp.URLBuilder, err = urlbuilder.New(apiOp.RequestCtx, &urlbuilder.DefaultPathResolver{
			Prefix: apiOp.URLPrefix,
		}, apiOp.Schemas)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	if apiOp.Schema == nil && apiOp.Schemas != nil {
		apiOp.Schema = apiOp.Schemas.LookupSchema(apiOp.Type)
	}

	if apiOp.Schema != nil {
		apiOp.Type = apiOp.Schema.ID
	}

	if apiOp.Schema != nil && apiOp.ErrorHandler != nil {
		apiOp.ErrorHandler = apiOp.Schema.ErrorHandler
	}

	if err := ValidateMethod(apiOp); err != nil {
		return err
	}

	return nil
}

func parseResponseFormat(c *app.RequestContext) string {
	//format := req.URL.Query().Get("_format")
	format := c.Query("_format")

	req := c.Request
	if format != "" {
		format = strings.TrimSpace(strings.ToLower(format))
	}

	/* Format specified */
	if allowedFormats[format] {
		return format
	}

	// User agent has Mozilla and browser accepts */*
	if IsBrowser(req, true) {
		return "html"
	}

	if isYaml(req) {
		return "yaml"
	}

	if isJSONL(req) {
		return "jsonl"
	}

	return "json"
}

func isYaml(req protocol.Request) bool {
	return strings.Contains(req.Header.Get("Accept"), "application/yaml")
}

func isJSONL(req protocol.Request) bool {
	return strings.Contains(req.Header.Get("Accept"), "application/jsonl")
}

//func parseMethod(req *http.Request) string {
//	method := req.URL.Query().Get("_method")
//	if method == "" {
//		method = req.Method
//	}
//	return method
//}

func Body(req protocol.Request) (types.APIObject, error) {
	form, err := req.MultipartForm()
	if err != nil {
		return types.APIObject{}, fmt.Errorf("error parsing multipart form: %s", err.Error())
	}

	if form != nil {
		return valuesToBody(form.Value), nil
	}

	return ReadBody(req)
	//if req.MultipartForm != nil {
	//	return valuesToBody(req.MultipartForm.Value), nil
	//}
	//
	//if req.PostForm != nil && len(req.PostForm) > 0 {
	//	return valuesToBody(map[string][]string(req.Form)), nil
	//}
	//
	//return ReadBody(req)
}

func valuesToBody(input map[string][]string) types.APIObject {
	result := map[string]interface{}{}
	for k, v := range input {
		result[k] = v
	}
	return toAPI(result)
}
