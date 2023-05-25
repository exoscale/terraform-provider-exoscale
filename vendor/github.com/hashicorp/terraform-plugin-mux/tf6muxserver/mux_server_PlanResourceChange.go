package tf6muxserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/internal/logging"
	"github.com/hashicorp/terraform-plugin-mux/internal/tf6dynamicvalue"
)

// PlanResourceChange calls the PlanResourceChange method, passing `req`, on
// the provider that returned the resource specified by req.TypeName in its
// schema.
func (s muxServer) PlanResourceChange(ctx context.Context, req *tfprotov6.PlanResourceChangeRequest) (*tfprotov6.PlanResourceChangeResponse, error) {
	rpc := "PlanResourceChange"
	ctx = logging.InitContext(ctx)
	ctx = logging.RpcContext(ctx, rpc)
	server, ok := s.resources[req.TypeName]

	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}

	ctx = logging.Tfprotov6ProviderServerContext(ctx, server)

	// Prevent ServerCapabilities.PlanDestroy from sending destroy plans to
	// servers which do not enable the capability.
	resourceCapabilities := s.resourceCapabilities[req.TypeName]

	if resourceCapabilities == nil || !resourceCapabilities.PlanDestroy {
		resourceSchema := s.resourceSchemas[req.TypeName]

		isDestroyPlan, err := tf6dynamicvalue.IsNull(resourceSchema, req.ProposedNewState)

		if err != nil {
			return nil, fmt.Errorf("unable to determine if request is destroy plan: %w", err)
		}

		if isDestroyPlan {
			logging.MuxTrace(ctx, "server does not enable destroy plans, returning without calling downstream server")

			resp := &tfprotov6.PlanResourceChangeResponse{
				// Presumably, we must preserve any prior private state so it
				// is still available during ApplyResourceChange.
				PlannedPrivate: req.PriorPrivate,
				PlannedState:   req.ProposedNewState,
			}

			return resp, nil
		}
	}

	logging.MuxTrace(ctx, "calling downstream server")

	return server.PlanResourceChange(ctx, req)
}
