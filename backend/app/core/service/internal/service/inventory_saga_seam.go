package service

import (
	"context"
)

// sagaSeam is the interface for cross-service SAGA orchestration that the
// inventory module would call when coordinating with procurement (SRM) or
// finance (AP). The real SAGA wiring (asynq task registration, po_id FK
// validation against the procurement service, compensation flows) is deferred
// to later modules. These stubs return success so the inventory vertical can
// compile and be exercised in isolation, and so unit tests can assert the
// seam is invoked without depending on services that do not exist yet.
//
// Each method takes the requesting context so a real implementation can
// propagate the viewer/tenant scope; the stub ignores it.

type sagaSeam interface {
	// validatePoReference would confirm that a purchase-order reference on an
	// inbound stock movement resolves to a real PO in the procurement service.
	validatePoReference(ctx context.Context, poId string) error
	// notifyProcurement would emit a SAGA step to the procurement service on a
	// stock event that affects an open PO (e.g. receipt completion).
	notifyProcurement(ctx context.Context, event stockEvent) error
}

type stockEvent struct {
	warehouseCode string
	skuCode       string
	delta         int64
}

// stubSagaSeam is the no-op implementation used until modules 2/3 land.
type stubSagaSeam struct{}

func (stubSagaSeam) validatePoReference(_ context.Context, _ string) error { return nil }
func (stubSagaSeam) notifyProcurement(_ context.Context, _ stockEvent) error { return nil }

var defaultSagaSeam sagaSeam = stubSagaSeam{}
