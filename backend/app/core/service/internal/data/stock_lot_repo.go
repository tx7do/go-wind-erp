package data

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/stocklocation"
	"go-wind-erp/app/core/service/internal/data/ent/stocklot"
	"go-wind-erp/app/core/service/internal/data/ent/stockmoveline"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockLotRepo 批次登记仓储（记录式批次/效期）。
//
// 批次余量不在本表存储，由 inv_stock_move_lines 按位置 usage 聚合推导：
// dest=INTERNAL 且 source≠INTERNAL 记 +qty（入库/盘盈/销退回货），
// source=INTERNAL 且 dest≠INTERNAL 记 −qty（出库/采退/盘亏），
// INTERNAL→INTERNAL 调拨净值 0。聚合在 Go 侧完成（镜像 aging 报表模式）。
type StockLotRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewStockLotRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *StockLotRepo {
	return &StockLotRepo{
		log:       ctx.NewLoggerHelper("stock_lot/repo/core-service"),
		entClient: entClient,
	}
}

// GetOrCreateTx 事务内按 (租户, SKU, 批号) get-or-create 批次。
// 已存在时忽略传入效期（以首次登记为准）；并发冲突由唯一索引兜底重查。
func (r *StockLotRepo) GetOrCreateTx(
	ctx context.Context,
	tx *ent.Tx,
	skuCode string,
	name string,
	expiry *time.Time,
) (uint32, error) {
	tid, _ := maybeTenantFromViewer(ctx)

	found, err := tx.StockLot.Query().
		Where(
			stocklot.TenantIDEQ(tid),
			stocklot.SkuCodeEQ(skuCode),
			stocklot.NameEQ(name),
		).
		Only(ctx)
	if err == nil && found != nil {
		return found.ID, nil
	}
	if err != nil && !ent.IsNotFound(err) {
		r.log.Errorf("query stock_lot failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("query stock_lot failed")
	}

	created, err := tx.StockLot.Create().
		SetTenantID(tid).
		SetSkuCode(skuCode).
		SetName(name).
		SetNillableExpiryDate(expiry).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		// 唯一索引冲突（并发同批号收货）→ 重查一次。
		if retry, rerr := tx.StockLot.Query().
			Where(
				stocklot.TenantIDEQ(tid),
				stocklot.SkuCodeEQ(skuCode),
				stocklot.NameEQ(name),
			).
			Only(ctx); rerr == nil && retry != nil {
			return retry.ID, nil
		}
		r.log.Errorf("insert stock_lot failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("insert stock_lot failed")
	}
	return created.ID, nil
}

// FindIdTx 按自然键查批次 ID，不存在返回 0（不报错——出库指派校验由服务层处理）。
func (r *StockLotRepo) FindIdTx(
	ctx context.Context,
	tx *ent.Tx,
	skuCode string,
	name string,
) (uint32, error) {
	tid, _ := maybeTenantFromViewer(ctx)

	row, err := tx.StockLot.Query().
		Where(
			stocklot.TenantIDEQ(tid),
			stocklot.SkuCodeEQ(skuCode),
			stocklot.NameEQ(name),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, nil
		}
		r.log.Errorf("query stock_lot failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("query stock_lot failed")
	}
	return row.ID, nil
}

// remainderOf 计算一批批次的剩余量（Go 侧聚合，镜像 aging 模式）。
// 接受 *ent.Client（主客户端或事务客户端均可）。
func (r *StockLotRepo) remainderOf(
	ctx context.Context,
	client *ent.Client,
	lotIDs []uint32,
) (map[uint32]int64, error) {
	remainders := make(map[uint32]int64, len(lotIDs))
	if len(lotIDs) == 0 {
		return remainders, nil
	}

	lines, err := client.StockMoveLine.Query().
		Where(stockmoveline.LotIDIn(lotIDs...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query move lines for lots failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("query move lines failed")
	}

	// 收集涉及的位置并一次性取 usage（位置表小，走 IN 单查）。
	locSet := map[uint32]struct{}{}
	for _, l := range lines {
		if l.SourceLocationID != nil {
			locSet[*l.SourceLocationID] = struct{}{}
		}
		if l.DestinationLocationID != nil {
			locSet[*l.DestinationLocationID] = struct{}{}
		}
	}
	locIDs := make([]uint32, 0, len(locSet))
	for id := range locSet {
		locIDs = append(locIDs, id)
	}
	locs, err := client.StockLocation.Query().
		Where(stocklocation.IDIn(locIDs...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query locations for lot aggregation failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("query locations failed")
	}
	usage := make(map[uint32]string, len(locs))
	for _, l := range locs {
		if l.Usage != nil {
			usage[l.ID] = string(*l.Usage)
		}
	}

	for _, l := range lines {
		if l.LotID == nil || *l.LotID == 0 || l.ExecutedQuantity == nil {
			continue
		}
		if l.SourceLocationID == nil || l.DestinationLocationID == nil {
			continue
		}
		srcInternal := usage[*l.SourceLocationID] == "INTERNAL"
		dstInternal := usage[*l.DestinationLocationID] == "INTERNAL"
		switch {
		case dstInternal && !srcInternal:
			remainders[*l.LotID] += *l.ExecutedQuantity
		case srcInternal && !dstInternal:
			remainders[*l.LotID] -= *l.ExecutedQuantity
		}
	}
	return remainders, nil
}

// LotRemaining 读取主客户端版本的批次余量（列表页/服务层校验用）。
func (r *StockLotRepo) LotRemaining(ctx context.Context, lotIDs []uint32) (map[uint32]int64, error) {
	return r.remainderOf(ctx, r.entClient.Client(), lotIDs)
}

// LotRemainingTx 事务内取单个批次余量（出库指派余量校验用）。
func (r *StockLotRepo) LotRemainingTx(ctx context.Context, tx *ent.Tx, lotID uint32) (int64, error) {
	remainders, err := r.remainderOf(ctx, tx.Client(), []uint32{lotID})
	if err != nil {
		return 0, err
	}
	return remainders[lotID], nil
}

// FefoLot FEFO 候选批次。
type FefoLot struct {
	LotID     uint32
	Name      string
	Expiry    *time.Time
	Remaining int64
}

// ListFefoTx 事务内取某 SKU 的可扣批次（remaining>0，效期升序 NULLS LAST）。
// PG ASC 默认 NULLS LAST——无批期的批次最后扣，符合"无期不先进"的直觉。
func (r *StockLotRepo) ListFefoTx(
	ctx context.Context,
	tx *ent.Tx,
	skuCode string,
) ([]FefoLot, error) {
	tid, _ := maybeTenantFromViewer(ctx)

	lots, err := tx.StockLot.Query().
		Where(
			stocklot.TenantIDEQ(tid),
			stocklot.SkuCodeEQ(skuCode),
		).
		Order(ent.Asc(stocklot.FieldExpiryDate)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query fefo lots failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("query fefo lots failed")
	}
	if len(lots) == 0 {
		return nil, nil
	}

	ids := make([]uint32, 0, len(lots))
	for _, l := range lots {
		ids = append(ids, l.ID)
	}
	remainders, err := r.remainderOf(ctx, tx.Client(), ids)
	if err != nil {
		return nil, err
	}

	result := make([]FefoLot, 0, len(lots))
	for _, l := range lots {
		if rem := remainders[l.ID]; rem > 0 {
			result = append(result, FefoLot{LotID: l.ID, Name: *l.Name, Expiry: l.ExpiryDate, Remaining: rem})
		}
	}
	return result, nil
}

// lotFilter PagingRequest 过滤字符串（前端 formValues JSON）。
type lotFilter struct {
	SkuCode   string `json:"skuCode"`
	LotStatus string `json:"lotStatus"`
}

// List 批次库存列表：批次登记行 + 聚合余量 + 效期状态。
// 状态（NORMAL/EXPIRING/EXPIRED）为推导值，过滤在 Go 侧完成（批次数
// 有限，先取全量再过滤分页，避免 SQL CASE builder 的脆弱性）。
func (r *StockLotRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockLotResponse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	var filter lotFilter
	if raw := req.GetFilter(); raw != "" {
		if err := json.Unmarshal([]byte(raw), &filter); err != nil {
			return nil, inventoryV1.ErrorBadRequest("invalid filter")
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	q := r.entClient.Client().StockLot.Query()
	if hasTenant {
		q = q.Where(stocklot.TenantIDEQ(tid))
	}
	if code := strings.TrimSpace(filter.SkuCode); code != "" {
		q = q.Where(stocklot.SkuCodeContainsFold(code))
	}

	rows, err := q.Order(ent.Desc(stocklot.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list stock lots failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("list stock lots failed")
	}

	ids := make([]uint32, 0, len(rows))
	for _, l := range rows {
		ids = append(ids, l.ID)
	}
	remainders, err := r.LotRemaining(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]*inventoryV1.StockLot, 0, len(rows))
	for _, l := range rows {
		remaining := remainders[l.ID]
		status := DeriveLotStatus(l.ExpiryDate, time.Now())
		if filter.LotStatus != "" && status.String() != filter.LotStatus {
			continue
		}

		item := &inventoryV1.StockLot{
			Id:                &l.ID,
			SkuCode:           l.SkuCode,
			Name:              l.Name,
			RemainingQuantity: &remaining,
			LotStatus:         &status,
		}
		if l.ExpiryDate != nil {
			item.ExpiryDate = timestamppb.New(*l.ExpiryDate)
		}
		if l.CreatedAt != nil {
			item.CreatedAt = timestamppb.New(*l.CreatedAt)
		}
		items = append(items, item)
	}

	total := uint64(len(items))
	// Go 侧分页（状态过滤后）。
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		items = []*inventoryV1.StockLot{}
	} else if start+pageSize < len(items) {
		items = items[start : start+pageSize]
	} else {
		items = items[start:]
	}

	return &inventoryV1.ListStockLotResponse{Total: total, Items: items}, nil
}

// DeriveLotStatus 按效期推导状态：无期=NORMAL；<now=EXPIRED；
// ≤now+30天=EXPIRING；否则 NORMAL。
func DeriveLotStatus(expiry *time.Time, now time.Time) inventoryV1.LotStatus {
	if expiry == nil {
		return inventoryV1.LotStatus_LOT_NORMAL
	}
	if !expiry.After(now) {
		return inventoryV1.LotStatus_LOT_EXPIRED
	}
	if !expiry.After(now.AddDate(0, 0, 30)) {
		return inventoryV1.LotStatus_LOT_EXPIRING
	}
	return inventoryV1.LotStatus_LOT_NORMAL
}
