package grpc

import (
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rakyll/statik/fs"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
	"google.golang.org/grpc"

	// Static files
	_ "github.com/zarbchain/zarb-go/www/grpc/statik"
)

type GatewayConfig struct {
	Enable     bool   `toml:"Enable" comment:"Enable gRPCGateway servers for client communication."`
	Address    string `toml:"Address"  comment:"Address to listen for incoming connections for gRPCGateway.Default port is 8080."`
	EnableCORS bool
}

// getOpenAPIHandler serves an OpenAPI UI.
// Adapted from https://github.com/philips/grpc-gateway-example/blob/a269bcb5931ca92be0ceae6130ac27ae89582ecc/cmd/serve.go#L63
func (s *Server) getOpenAPIHandler() (http.Handler, error) {
	err := mime.AddExtensionType(".svg", "image/svg+xml")
	if err != nil {
		return nil, err
	}

	statikFS, err := fs.New()
	if err != nil {
		return nil, err
	}

	return http.FileServer(statikFS), nil
}

func (s *Server) startGateway() error {
	if !s.config.Gateway.Enable {
		return nil
	}

	conn, err := grpc.DialContext(
		s.ctx,
		s.config.Address,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}

	gwmux := runtime.NewServeMux()
	err = zarb.RegisterZarbHandler(s.ctx, gwmux, conn)
	if err != nil {
		return err
	}

	oa, err := s.getOpenAPIHandler()
	if err != nil {
		return err
	}

	gwServer := &http.Server{
		Addr: s.config.Gateway.Address,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api") {
				gwmux.ServeHTTP(w, r)
				return
			}
			oa.ServeHTTP(w, r)
		}),
	}
	if s.config.Gateway.EnableCORS {
		gwServer.Handler = allowCORS(gwServer.Handler)
	}
	return gwServer.ListenAndServe()
}

// preflightHandler adds the necessary headers in order to serve
// CORS from any origin using the methods "GET", "HEAD", "POST", "PUT", "DELETE"
// We insist, don't do this without consideration in production systems.
func preflightHandler(w http.ResponseWriter) {
	headers := []string{"Content-Type", "Accept", "Authorization"}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
}

// allowCORS allows Cross Origin Resoruce Sharing from any origin.
// Don't do this without consideration in production systems.
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
