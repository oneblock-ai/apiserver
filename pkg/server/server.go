package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/sirupsen/logrus"

	"github.com/rancher/apiserver/pkg/builtin"
	"github.com/rancher/apiserver/pkg/handlers"
	"github.com/rancher/apiserver/pkg/metrics"
	"github.com/rancher/apiserver/pkg/parse"
	"github.com/rancher/apiserver/pkg/subscribe"
	"github.com/rancher/apiserver/pkg/types"
	"github.com/rancher/apiserver/pkg/writer"

	"github.com/rancher/wrangler/v3/pkg/schemas/validation"
)

type RequestHandler interface {
	http.Handler

	GetSchemas() *types.APISchemas
	Handle(apiOp *types.APIRequest)
}

type Server struct {
	ResponseWriters map[string]types.ResponseWriter
	Schemas         *types.APISchemas
	AccessControl   types.AccessControl
	Parser          parse.Parser
	URLParser       parse.URLParser2
}

func DefaultAPIServer() *Server {
	s := &Server{
		Schemas: types.EmptyAPISchemas().MustAddSchemas(builtin.Schemas),
		ResponseWriters: map[string]types.ResponseWriter{
			"json": &writer.GzipWriter{
				ResponseWriter: &writer.EncodingResponseWriter{
					ContentType: "application/json",
					Encoder:     types.JSONEncoder,
				},
			},
			"jsonl": &writer.GzipWriter{
				ResponseWriter: &writer.EncodingResponseWriter{
					ContentType: "application/jsonl",
					Encoder:     types.JSONLinesEncoder,
				},
			},
			"html": &writer.GzipWriter{
				ResponseWriter: &writer.HTMLResponseWriter{
					EncodingResponseWriter: writer.EncodingResponseWriter{
						Encoder:     types.JSONEncoder,
						ContentType: "application/json",
					},
				},
			},
			"yaml": &writer.GzipWriter{
				ResponseWriter: &writer.EncodingResponseWriter{
					ContentType: "application/yaml",
					Encoder:     types.YAMLEncoder,
				},
			},
		},
		AccessControl: &SchemaBasedAccess{},
		Parser:        parse.Parse,
		//URLParser:     parse.MuxURLParser,
		URLParser: parse.HertzURLParser,
	}

	subscribe.Register(s.Schemas, subscribe.DefaultGetter, os.Getenv("SERVER_VERSION"))
	return s
}

func (s *Server) setDefaults(ctx *types.APIRequest) {
	if ctx.ResponseWriter == nil {
		ctx.ResponseWriter = s.ResponseWriters[ctx.ResponseFormat]
		if ctx.ResponseWriter == nil {
			ctx.ResponseWriter = s.ResponseWriters["json"]
		}
	}

	if ctx.ErrorHandler == nil {
		ctx.ErrorHandler = handlers.ErrorHandler
	}

	ctx.AccessControl = s.AccessControl

	if ctx.Schemas == nil {
		ctx.Schemas = s.Schemas
	}
}

func (s *Server) HandlerFunc(ctx context.Context, c *app.RequestContext) {
	s.Handle(&types.APIRequest{
		Context:    ctx,
		RequestCtx: c,
		//ProtocolRequest:  c.Request,
		//ProtocolResponse: c.Response,
	})
}

//func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
//	s.Handle(&types.APIRequest{
//		//Request:  req,
//		//Response: rw,
//	})
//}

func (s *Server) Handle(apiOp *types.APIRequest) {
	s.handle(apiOp, s.Parser)
}

func (s *Server) handle(apiOp *types.APIRequest, parser parse.Parser) {
	if apiOp.Schemas == nil {
		apiOp.Schemas = s.Schemas
	}

	logrus.Infof("Handling request, %+v", apiOp.Schemas)
	//if err := parser(apiOp, parse.MuxURLParser); err != nil {
	if err := parser(apiOp, parse.HertzURLParser); err != nil {
		// ensure defaults set so writer is assigned
		s.setDefaults(apiOp)
		apiOp.WriteError(err)
		return
	}

	s.setDefaults(apiOp)
	fmt.Printf("Handling request 2, %+v\n", apiOp.Schemas.Schemas)

	var cloned *types.APISchemas
	for id, schema := range apiOp.Schemas.Schemas {
		fmt.Printf("handling request 3, %v\n", schema.RequestModifier)
		if schema.RequestModifier == nil {
			continue
		}

		if cloned == nil {
			cloned = apiOp.Schemas.ShallowCopy()
		}

		schema := schema.DeepCopy()
		fmt.Printf("handling request 4, %+v\n", schema)
		schema = schema.RequestModifier(apiOp, schema)
		cloned.Schemas[id] = schema
	}

	if cloned != nil {
		apiOp.Schemas = cloned
	}

	if apiOp.Schema != nil && apiOp.Schema.RequestModifier != nil {
		apiOp.Schema = apiOp.Schema.RequestModifier(apiOp, apiOp.Schema)
	}

	requestStart := time.Now()
	var code int
	var data interface{}
	var err error
	if code, data, err = s.handleOp(apiOp); err != nil {
		apiOp.WriteError(err)
	} else if obj, ok := data.(types.APIObject); ok {
		apiOp.WriteResponse(code, obj)
	} else if list, ok := data.(types.APIObjectList); ok {
		apiOp.WriteResponseList(code, list)
	} else if code > http.StatusOK {
		apiOp.RequestCtx.Status(code)
	}

	metrics.RecordResponseTime(apiOp.Type, apiOp.Method, strconv.Itoa(code),
		float64(time.Since(requestStart).Milliseconds()))
}

func (s *Server) handleOp(apiOp *types.APIRequest) (int, interface{}, error) {
	if err := CheckCSRF(apiOp); err != nil {
		return 0, nil, err
	}

	if apiOp.Schema == nil {
		return http.StatusNotFound, nil, nil
	}

	action, err := ValidateAction(apiOp)
	if err != nil {
		return 0, nil, err
	}

	if action != nil {
		if apiOp.Name != "" {
			data, err := handle(apiOp, apiOp.Schema.ByIDHandler, handlers.ByIDHandler)
			if err != nil {
				return http.StatusOK, data, err
			}
		}
		return http.StatusOK, nil, handleAction(apiOp)
	}

	switch apiOp.Method {
	case http.MethodGet:
		if apiOp.Name == "" {
			data, err := handleList(apiOp, apiOp.Schema.ListHandler,
				handlers.MetricsListHandler("200", handlers.ListHandler))
			return http.StatusOK, data, err
		}
		data, err := handle(apiOp, apiOp.Schema.ByIDHandler,
			handlers.MetricsHandler("200", handlers.ByIDHandler))
		return http.StatusOK, data, err
	case http.MethodPatch:
		fallthrough
	case http.MethodPut:
		data, err := handle(apiOp, apiOp.Schema.UpdateHandler,
			handlers.MetricsHandler("200", handlers.UpdateHandler))
		return http.StatusOK, data, err
	case http.MethodPost:
		data, err := handle(apiOp, apiOp.Schema.CreateHandler,
			handlers.MetricsHandler("201", handlers.CreateHandler))
		return http.StatusCreated, data, err
	case http.MethodDelete:
		data, err := handle(apiOp, apiOp.Schema.DeleteHandler,
			handlers.MetricsHandler("200", handlers.DeleteHandler))
		return http.StatusOK, data, err
	}

	return http.StatusNotFound, nil, nil
}

func handleList(apiOp *types.APIRequest, custom types.RequestListHandler,
	handler types.RequestListHandler) (types.APIObjectList, error) {
	if custom != nil {
		return custom(apiOp)
	}
	return handler(apiOp)
}

func handle(apiOp *types.APIRequest, custom types.RequestHandler,
	handler types.RequestHandler) (types.APIObject, error) {
	if custom != nil {
		return custom(apiOp)
	}
	return handler(apiOp)
}

func handleAction(context *types.APIRequest) error {
	if err := context.AccessControl.CanAction(context, context.Schema, context.Action); err != nil {
		return err
	}
	if handler, ok := context.Schema.ActionHandlers[context.Action]; ok {
		//handler.ServeHTTP(context.Response, context.Request)
		fmt.Println("handleAction", handler)
		return validation.ErrComplete
	}
	return nil
}

func (s *Server) CustomAPIUIResponseWriter(cssURL, jsURL, version writer.StringGetter) {
	wi, ok := s.ResponseWriters["html"]
	if !ok {
		return
	}
	gw, ok := wi.(*writer.GzipWriter)
	if !ok {
		return
	}

	w, ok := gw.ResponseWriter.(*writer.HTMLResponseWriter)
	if !ok {
		return
	}
	w.CSSURL = cssURL
	w.JSURL = jsURL
	w.APIUIVersion = version
}
