package main

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/network/standard"

	"github.com/rancher/apiserver/pkg/server"
	"github.com/rancher/apiserver/pkg/store/apiroot"
	"github.com/rancher/apiserver/pkg/store/empty"
	"github.com/rancher/apiserver/pkg/types"

	hServer "github.com/cloudwego/hertz/pkg/app/server"
)

type Foo struct {
	Bar string `json:"bar"`
}

type FooStore struct {
	empty.Store
}

func (f *FooStore) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	return types.APIObject{
		Type: "foos",
		ID:   id,
		Object: Foo{
			Bar: "baz",
		},
	}, nil
}

func (f *FooStore) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	return types.APIObjectList{
		Objects: []types.APIObject{
			{
				Type: "foostore",
				ID:   "foo",
				Object: Foo{
					Bar: "baz",
				},
			},
		},
	}, nil
}

func main() {
	// Create the default server
	s := server.DefaultAPIServer()

	// Add some types to it and setup the store and supported methods
	s.Schemas.MustImportAndCustomize(Foo{}, func(schema *types.APISchema) {
		schema.Store = &FooStore{}
		schema.CollectionMethods = []string{http.MethodGet}
		schema.ResourceMethods = []string{http.MethodGet}
	})

	// Register root handler to list api versions
	apiroot.Register(s.Schemas, []string{"v1"})

	// Setup mux router to assign variables the server will look for (refer to MuxURLParser for all variable names)
	//router := mux.NewRouter()
	//router.Handle("/{prefix}/{type}", s)
	//router.Handle("/{prefix}/{type}/{name}", s)
	//
	//// When a route is found construct a custom API request to serves up the API root content
	//router.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
	//	s.Handle(&types.APIRequest{
	//		Request:   r,
	//		Response:  rw,
	//		Type:      "apiRoot",
	//		URLPrefix: "v1",
	//	})
	//})

	//h := hServer.Default(hServer.WithHostPorts("127.0.0.1:8888"))
	h := hServer.New(hServer.WithTransport(standard.NewTransporter))

	h.Any(":prefix/:type", s.HandlerFunc)
	h.Any(":prefix/:type/:name", s.HandlerFunc)
	h.NoRoute(func(ctx context.Context, c *app.RequestContext) {
		s.Handle(&types.APIRequest{
			RequestCtx: c,
			Type:       "apiRoot",
			URLPrefix:  "v1",
		})
	})
	h.Spin()

	// Start API Server
	//log.Print("Listening on :8088")
	//log.Fatal(http.ListenAndServe(":8088", router))
}
