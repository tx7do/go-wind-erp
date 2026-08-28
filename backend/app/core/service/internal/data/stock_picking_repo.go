package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"
	"go-wind-erp/app/core/service/internal/data/ent/stockpicking"
	"go-wind-erp/app/core/service/internal/data/ent/stockmove"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockPickingRepo 拣货单仓储（借鉴 Odoo stock.picking：一等文档，有生命周期）。
// 不存储 state——派生态从子 moves 每次读取时聚合计算（见 Get）。
type StockPickingRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	pickingTypeConverter *mapper.EnumTypeConverter[inventoryV1.StockPicking_PickingType, stockpicking.PickingType]
	moveStateConverter   *mapper.EnumTypeConverter[inventoryV1.StockMove_State, stockmove.State]

	mapper *mapper.CopierMapper[inventoryV1.StockPicking, ent.StockPicking]

	repository *entCrud.Repository[
		ent.StockPickingQuery, ent.StockPickingSelect,
		ent.StockPickingCreate, ent.StockPickingCreateBulk,
		ent.StockPickingUpdate, ent.StockPickingUpdateOne,
		ent.StockPickingDelete,
		predicate.StockPicking,
		inventoryV1.StockPicking, ent.StockPicking,
	]

	moveRepo *StockMoveRepo
}

func NewStockPickingRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	moveRepo *StockMoveRepo,
) *StockPickingRepo {
	repo := &StockPickingRepo{
		log:       ctx.NewLoggerHelper("stock_picking/repo/core-service"),
		entClient: entClient,
		moveRepo:  moveRepo,
	}

	repo.init()

	return repo
}

func (r *StockPickingRepo) init() {
	r.mapper = mapper.NewCopierMapper[inventoryV1.StockPicking, ent.StockPicking]()
	r.pickingTypeConverter = mapper.NewEnumTypeConverter[inventoryV1.StockPicking_PickingType, stockpicking.PickingType](inventoryV1.StockPicking_PickingType_name, inventoryV1.StockPicking_PickingType_value)
	r.moveStateConverter = mapper.NewEnumTypeConverter[inventoryV1.StockMove_State, stockmove.State](inventoryV1.StockMove_State_name, inventoryV1.StockMove_State_value)

	r.repository = entCrud.NewRepository[
		ent.StockPickingQuery, ent.StockPickingSelect,
		ent.StockPickingCreate, ent.StockPickingCreateBulk,
		ent.StockPickingUpdate, ent.StockPickingUpdateOne,
		ent.StockPickingDelete,
		predicate.StockPicking,
		inventoryV1.StockPicking, ent.StockPicking,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.pickingTypeConverter.NewConverterPair())
}

func (r *StockPickingRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().StockPicking.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *StockPickingRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockPickingResponse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockPicking.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &inventoryV1.ListStockPickingResponse{Total: 0, Items: nil}, nil
	}

	// 为每个 picking 计算派生态（从子 moves 聚合，不存储）。
	for _, p := range ret.Items {
		ds := r.computeDerivedState(ctx, p.GetId())
		p.DerivedState = &ds
	}

	return &inventoryV1.ListStockPickingResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *StockPickingRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().StockPicking.Query().
		Where(stockpicking.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, inventoryV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// GetOutgoingPickingIDs 取所有出库（OUTGOING）拣货单的 ID，供 COGS
// 报表按月聚合 move-line 成本用。
func (r *StockPickingRepo) GetOutgoingPickingIDs(ctx context.Context) ([]uint32, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	q := r.entClient.Client().StockPicking.Query().
		Where(stockpicking.PickingTypeEQ(stockpicking.PickingTypeOutgoing))
	if hasTenant {
		q.Where(stockpicking.TenantIDEQ(tid))
	}
	ids, err := q.IDs(ctx)
	if err != nil {
		r.log.Errorf("query outgoing picking ids failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("query outgoing picking ids failed")
	}
	return ids, nil
}

func (r *StockPickingRepo) Get(ctx context.Context, req *inventoryV1.GetStockPickingRequest) (*inventoryV1.StockPicking, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockPicking.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *inventoryV1.GetStockPickingRequest_Id:
		whereCond = append(whereCond, stockpicking.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}
	if dto == nil {
		return dto, nil
	}

	// 组装子 moves 并计算派生态。
	dto.Moves = r.moveRepo.GetMovesByPicking(ctx, req.GetId())
	ds := r.deriveStateFromMoves(dto.Moves)
	dto.DerivedState = &ds
	return dto, nil
}

func (r *StockPickingRepo) Create(ctx context.Context, req *inventoryV1.CreateStockPickingRequest) (*inventoryV1.StockPicking, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	// source/dest location 必须由服务层按 picking_type + 仓库推导后设置。
	if req.Data.SourceLocationId == nil || req.Data.DestinationLocationId == nil {
		return nil, inventoryV1.ErrorBadRequest("source and destination locations are required")
	}

	pickingNumber := "PK" + fmt.Sprintf("%d", time.Now().UnixMilli())

	builder := r.entClient.Client().StockPicking.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetPickingNumber(pickingNumber).
		SetNillablePickingType(r.pickingTypeConverter.ToEntity(req.Data.PickingType)).
		SetNillableSourceLocationID(req.Data.SourceLocationId).
		SetNillableDestinationLocationID(req.Data.DestinationLocationId).
		SetNillablePurchaseOrderID(req.Data.PurchaseOrderId).
		SetNillablePartnerCode(req.Data.PartnerCode).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert stock_picking failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("insert stock_picking failed")
	}

	// 创建嵌套子 moves（状态恒为 DRAFT，location 从拣货单继承）。
	if err := r.moveRepo.CreateMovesForPicking(
		ctx, t.ID,
		*t.SourceLocationID, *t.DestinationLocationID,
		req.Data.Moves,
	); err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(t)
	dto.Moves = r.moveRepo.GetMovesByPicking(ctx, t.ID)
	ds := r.deriveStateFromMoves(dto.Moves)
	dto.DerivedState = &ds
	return dto, nil
}

func (r *StockPickingRepo) Delete(ctx context.Context, req *inventoryV1.DeleteStockPickingRequest) error {
	if req == nil {
		return inventoryV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().StockPicking.Delete()
	delBuilder.Where(stockpicking.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(stockpicking.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return inventoryV1.ErrorNotFound("stock_picking not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return inventoryV1.ErrorInternalServerError("delete failed")
	}

	// 级联删除子 moves。
	if err := r.moveRepo.DeleteMovesByPicking(ctx, req.GetId()); err != nil {
		r.log.Errorf("delete child moves failed: %s", err.Error())
	}

	return nil
}

// GetConfirmedMovesTx 取拣货单中 CONFIRMED 状态的子移动（供
// StockPickingService.Validate 在事务内执行）。委托给 moveRepo。
func (r *StockPickingRepo) GetConfirmedMovesTx(
	ctx context.Context,
	tx *ent.Tx,
	pickingID uint32,
) ([]*inventoryV1.StockMove, error) {
	return r.moveRepo.GetConfirmedMovesByPickingTx(ctx, tx, pickingID)
}

// CancelMovesTx 将拣货单内所有非终态 moves 置为 CANCELLED（供
// StockPickingService.Cancel 使用）。委托给 moveRepo。
func (r *StockPickingRepo) CancelMovesTx(
	ctx context.Context,
	tx *ent.Tx,
	pickingID uint32,
) error {
	if err := r.moveRepo.CancelDraftMovesTx(ctx, tx, pickingID); err != nil {
		return err
	}
	return r.moveRepo.CancelConfirmedMovesTx(ctx, tx, pickingID)
}

// TransitionMoveStateTx 事务内原子状态迁移（委托给 moveRepo）。
func (r *StockPickingRepo) TransitionMoveStateTx(
	ctx context.Context,
	tx *ent.Tx,
	id uint32,
	from, to inventoryV1.StockMove_State,
) error {
	return r.moveRepo.TransitionMoveStateTx(ctx, tx, id, from, to)
}

// BeginTx 开启一个 DB 事务（供 Validate 在事务内执行）。
func (r *StockPickingRepo) BeginTx(ctx context.Context) (tx *ent.Tx, err error) {
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("start transaction failed")
	}
	return tx, nil
}

func (r *StockPickingRepo) FinishTx(tx *ent.Tx, err error) {
	if tx == nil {
		return
	}
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
		}
		return
	}
	if commitErr := tx.Commit(); commitErr != nil {
		r.log.Errorf("transaction commit failed: %s", commitErr.Error())
	}
}

// computeDerivedState 从 DB 读取子 moves 并计算派生态（用于 List，每次读取时算）。
func (r *StockPickingRepo) computeDerivedState(ctx context.Context, pickingID uint32) inventoryV1.StockPicking_DerivedState {
	moves := r.moveRepo.GetMovesByPicking(ctx, pickingID)
	return r.deriveStateFromMoves(moves)
}

// deriveStateFromMoves 按子 move 状态聚合派生态（借鉴 Odoo _compute_state，
// 简化为无 waiting/assigned/partially_available）：
//   - 任何 DRAFT move → DRAFT
//   - 全部 DONE → DONE
//   - 全部 CANCELLED → CANCELLED
//   - 否则 → CONFIRMED
func (r *StockPickingRepo) deriveStateFromMoves(moves []*inventoryV1.StockMove) inventoryV1.StockPicking_DerivedState {
	if len(moves) == 0 {
		return inventoryV1.StockPicking_DRAFT
	}
	hasDraft := false
	allDone := true
	allCancelled := true
	for _, m := range moves {
		if m == nil {
			continue
		}
		switch m.GetState() {
		case inventoryV1.StockMove_DRAFT:
			hasDraft = true
			allDone = false
			allCancelled = false
		case inventoryV1.StockMove_DONE:
			allCancelled = false
		case inventoryV1.StockMove_CANCELLED:
			allDone = false
		case inventoryV1.StockMove_CONFIRMED:
			allDone = false
			allCancelled = false
		}
	}
	if hasDraft {
		return inventoryV1.StockPicking_DRAFT
	}
	if allDone {
		return inventoryV1.StockPicking_DONE
	}
	if allCancelled {
		return inventoryV1.StockPicking_CANCELLED
	}
	return inventoryV1.StockPicking_CONFIRMED
}
