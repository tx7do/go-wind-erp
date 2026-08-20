package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/inventory"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

type InventoryRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper        *mapper.CopierMapper[inventoryV1.Inventory, ent.Inventory]
	statusConverter *mapper.EnumTypeConverter[inventoryV1.Inventory_Status, inventory.Status]

	repository *entCrud.Repository[
		ent.InventoryQuery, ent.InventorySelect,
		ent.InventoryCreate, ent.InventoryCreateBulk,
		ent.InventoryUpdate, ent.InventoryUpdateOne,
		ent.InventoryDelete,
		predicate.Inventory,
		inventoryV1.Inventory, ent.Inventory,
	]
}

func NewInventoryRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *InventoryRepo {
	repo := &InventoryRepo{
		log:           ctx.NewLoggerHelper("inventory/repo/core-service"),
		entClient:     entClient,
	}

	repo.init()

	return repo
}

func (r *InventoryRepo) init() {
	r.mapper = mapper.NewCopierMapper[inventoryV1.Inventory, ent.Inventory]()
	r.statusConverter = mapper.NewEnumTypeConverter[inventoryV1.Inventory_Status, inventory.Status](inventoryV1.Inventory_Status_name, inventoryV1.Inventory_Status_value)

	r.repository = entCrud.NewRepository[
		ent.InventoryQuery, ent.InventorySelect,
		ent.InventoryCreate, ent.InventoryCreateBulk,
		ent.InventoryUpdate, ent.InventoryUpdateOne,
		ent.InventoryDelete,
		predicate.Inventory,
		inventoryV1.Inventory, ent.Inventory,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *InventoryRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Inventory.Query()
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

func (r *InventoryRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListInventoryResponse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Inventory.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &inventoryV1.ListInventoryResponse{Total: 0, Items: nil}, nil
	}

	return &inventoryV1.ListInventoryResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *InventoryRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Inventory.Query().
		Where(inventory.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, inventoryV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *InventoryRepo) Get(ctx context.Context, req *inventoryV1.GetInventoryRequest) (*inventoryV1.Inventory, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Inventory.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *inventoryV1.GetInventoryRequest_Id:
		whereCond = append(whereCond, inventory.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// SumQuantity 库存总量（所有记录 quantity 之和）。空表时 Sum 无结果行，兜底为 0。
func (r *InventoryRepo) SumQuantity(ctx context.Context) (int64, error) {
	var totals []int64
	if err := r.entClient.Client().Inventory.Query().
		Aggregate(ent.Sum(inventory.FieldQuantity)).
		Scan(ctx, &totals); err != nil {
		r.log.Errorf("sum quantity failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("sum quantity failed")
	}
	if len(totals) == 0 {
		return 0, nil
	}
	return totals[0], nil
}

// CountDistinctSku 在库 SKU 数（按 sku_code 去重计数）。
func (r *InventoryRepo) CountDistinctSku(ctx context.Context) (int, error) {
	var distinctSkus []string
	if err := r.entClient.Client().Inventory.Query().
		GroupBy(inventory.FieldSkuCode).
		Scan(ctx, &distinctSkus); err != nil {
		r.log.Errorf("count distinct sku failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("count distinct sku failed")
	}
	return len(distinctSkus), nil
}

// ListLowStock 低库存清单：quantity < threshold，按数量升序，最多 limit 条。
func (r *InventoryRepo) ListLowStock(ctx context.Context, threshold int64, limit int) ([]*inventoryV1.Inventory, error) {
	rows, err := r.entClient.Client().Inventory.Query().
		Where(inventory.QuantityLT(threshold)).
		Order(ent.Asc(inventory.FieldQuantity)).
		Limit(limit).
		All(ctx)
	if err != nil {
		r.log.Errorf("list low stock failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("list low stock failed")
	}

	items := make([]*inventoryV1.Inventory, 0, len(rows))
	for _, row := range rows {
		items = append(items, r.mapper.ToDTO(row))
	}
	return items, nil
}

func (r *InventoryRepo) Create(ctx context.Context, req *inventoryV1.CreateInventoryRequest) (*inventoryV1.Inventory, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Inventory.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableWarehouseCode(req.Data.WarehouseCode).
		SetNillableSkuCode(req.Data.SkuCode).
		SetNillableQuantity(req.Data.Quantity).
		SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert inventory failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("insert inventory failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *InventoryRepo) Update(ctx context.Context, req *inventoryV1.UpdateInventoryRequest) (*inventoryV1.Inventory, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &inventoryV1.CreateInventoryRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().Inventory.UpdateOneID(req.GetId())
	builder.Where(inventory.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(inventory.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *inventoryV1.Inventory) {
			builder.
				SetNillableWarehouseCode(req.Data.WarehouseCode).
				SetNillableSkuCode(req.Data.SkuCode).
				SetNillableQuantity(req.Data.Quantity).
				SetNillableStatus(r.statusConverter.ToEntity(req.Data.Status)).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			// updated_by 强制由服务端 viewer context 推导，忽略客户端传入值
			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(inventory.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *InventoryRepo) Delete(ctx context.Context, req *inventoryV1.DeleteInventoryRequest) error {
	if req == nil {
		return inventoryV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().Inventory.Delete()
	delBuilder.Where(inventory.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(inventory.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return inventoryV1.ErrorNotFound("inventory not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return inventoryV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
