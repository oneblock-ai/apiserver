package writer

import (
	"github.com/rancher/apiserver/pkg/types"
)

func AddCommonResponseHeader(apiOp *types.APIRequest) {
	addExpires(apiOp)
	addSchemasHeader(apiOp)
}

func addSchemasHeader(apiOp *types.APIRequest) {
	schema := apiOp.Schemas.Schemas["schema"]
	if schema == nil {
		return
	}

	apiOp.RequestCtx.Response.Header.Set("X-Api-Schemas", apiOp.URLBuilder.Collection(schema))
}

func addExpires(apiOp *types.APIRequest) {
	apiOp.RequestCtx.Response.Header.Set("Expires", "Wed 24 Feb 1982 18:42:00 GMT")
}
