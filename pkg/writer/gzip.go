package writer

import (
	"compress/gzip"
	"io"
	"strings"

	"github.com/rancher/apiserver/pkg/types"
)

type GzipWriter struct {
	types.ResponseWriter
}

func setup(apiOp *types.APIRequest) (*types.APIRequest, io.Closer) {
	resp := apiOp.RequestCtx.GetResponse()
	if !strings.Contains(resp.Header.Get("Accept-Encoding"), "gzip") {
		return apiOp, io.NopCloser(nil)
	}

	resp.Header.Set("Content-Encoding", "gzip")
	resp.Header.Del("Content-Length")

	gz := gzip.NewWriter(resp.BodyWriter())
	//gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: resp.BodyWriter()}
	//gzw := &gzipResponseWriter{Writer: gz}

	newOp := *apiOp
	newOp.RequestCtx.Response = *resp
	return &newOp, gz
}

func (g *GzipWriter) Write(apiOp *types.APIRequest, code int, obj types.APIObject) {
	apiOp, closer := setup(apiOp)
	defer closer.Close()
	g.ResponseWriter.Write(apiOp, code, obj)
}

func (g *GzipWriter) WriteList(apiOp *types.APIRequest, code int, obj types.APIObjectList) {
	apiOp, closer := setup(apiOp)
	defer closer.Close()
	g.ResponseWriter.WriteList(apiOp, code, obj)
}

type gzipResponseWriter struct {
	io.Writer
	//http.ResponseWriter
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}
