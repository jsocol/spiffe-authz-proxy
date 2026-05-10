package envoyauthzserver

import (
	"context"
	"fmt"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

func (s *Server) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	resp := &authv3.CheckResponse{}

	source := req.GetAttributes().GetSource()
	spid, err := spiffeid.FromString(source.GetPrincipal())
	if err != nil {
		resp.HttpResponse = &authv3.CheckResponse_ErrorResponse{
			ErrorResponse: &authv3.DeniedHttpResponse{
				Body: fmt.Sprintf("source principal was not a spiffeid: %s", err),
			},
		}

		return resp, nil
	}

	httpReq := req.GetAttributes().GetRequest().GetHttp()
	path := httpReq.GetPath()
	method := httpReq.GetMethod()

	err = s.authz.Authorize(ctx, spid, method, path)
	if err != nil {
		resp.HttpResponse = &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typev3.HttpStatus{
					Code: typev3.StatusCode_Forbidden,
				},
				Body: err.Error(),
			},
		}

		return resp, nil
	}

	resp.HttpResponse = &authv3.CheckResponse_OkResponse{}

	return resp, nil
}
