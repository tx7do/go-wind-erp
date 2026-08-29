package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

// 收款审批联动约定：biz_type="RECEIPT"，biz_ref="RECEIPT:{id}"。
const (
	receiptApprovalBizType = "RECEIPT"
	receiptApprovalBizRef  = "RECEIPT:%d"
)

// ReceiptService 收款服务（Create=提交收款申请，经审批后入账）
type ReceiptService struct {
	financeV1.UnimplementedReceiptServiceServer

	log *log.Helper

	receiptRepo         *data.ReceiptRepo
	receivableRepo      *data.ReceivableRepo
	approvalRequestRepo *data.ApprovalRequestRepo
	journal             *journalPoster
}

func NewReceiptService(
	ctx *bootstrap.Context,
	receiptRepo *data.ReceiptRepo,
	receivableRepo *data.ReceivableRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
	journal *journalPoster,
) *ReceiptService {
	svc := &ReceiptService{
		log:                 ctx.NewLoggerHelper("receipt/service/core-service"),
		receiptRepo:         receiptRepo,
		receivableRepo:      receivableRepo,
		approvalRequestRepo: approvalRequestRepo,
		journal:             journal,
	}

	return svc
}

func (s *ReceiptService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListReceiptResponse, error) {
	return s.receiptRepo.List(ctx, req)
}

func (s *ReceiptService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.CountReceiptResponse, error) {
	count, err := s.receiptRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &financeV1.CountReceiptResponse{Count: uint64(count)}, nil
}

func (s *ReceiptService) Get(ctx context.Context, req *financeV1.GetReceiptRequest) (*financeV1.Receipt, error) {
	return s.receiptRepo.Get(ctx, req)
}

// Create 提交收款申请：落 PENDING 收款记录并自动生成审批单
// （biz_type="RECEIPT"，biz_ref="RECEIPT:{id}"）。审批通过后经
// [ApplyApproved] 真正入账（累计应收单）；驳回则 [RejectApplied]。
// 预校验仅做只读检查（防呆）；权威的防超收/状态守卫在入账时原子执行。
func (s *ReceiptService) Create(ctx context.Context, req *financeV1.CreateReceiptRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}
	if req.Data.GetReceivableId() == 0 {
		return nil, financeV1.ErrorBadRequest("receivable_id is required")
	}
	amount := req.Data.GetAmount()
	if amount <= 0 {
		return nil, financeV1.ErrorBadRequest("amount must be positive")
	}

	receivable, err := s.receivableRepo.Get(ctx, &financeV1.GetReceivableRequest{
		QueryBy: &financeV1.GetReceivableRequest_Id{Id: req.Data.GetReceivableId()},
	})
	if err != nil {
		return nil, err
	}
	if _, overflow := addChecked(receivable.GetPaidAmount(), amount); overflow {
		return nil, financeV1.ErrorBadRequest("paid amount overflow")
	}
	// 只读预检：明显超收直接拒绝（竞态仍由入账时的 SQL 条件兜底）。
	if receivable.GetPaidAmount()+amount > receivable.GetAmount() {
		return nil, financeV1.ErrorBadRequest("receipt exceeds receivable amount")
	}

	receipt, err := s.receiptRepo.Create(ctx, &financeV1.CreateReceiptRequest{
		Data: &financeV1.Receipt{
			ReceivableId: req.Data.ReceivableId,
			Amount:       req.Data.Amount,
			Method:       req.Data.Method,
			Remark:       req.Data.Remark,
		},
	})
	if err != nil {
		return nil, err
	}

	// 生成收款审批单；失败不影响收款记录（PENDING 可重提审批）。
	if _, aerr := s.approvalRequestRepo.Create(ctx, &approvalV1.CreateApprovalRequestRequest{
		Data: &approvalV1.ApprovalRequest{
			Title:   trans.Ptr(fmt.Sprintf("收款申请 %s（%d 分 → 应收 %s）", receipt.GetReceiptNumber(), amount, receivable.GetReceivableNumber())),
			BizType: trans.Ptr(receiptApprovalBizType),
			BizRef:  trans.Ptr(fmt.Sprintf(receiptApprovalBizRef, receipt.GetId())),
			Summary: trans.Ptr(fmt.Sprintf("客户 %s；应收 %d 分，已收 %d 分", receivable.GetCustomerCode(), receivable.GetAmount(), receivable.GetPaidAmount())),
		},
	}); aerr != nil {
		s.log.Errorf("create approval for receipt %d failed: %s", receipt.GetId(), aerr.Error())
	}

	return &emptypb.Empty{}, nil
}

// ApplyApproved 审批通过后入账：PENDING→APPLIED（条件迁移）后原子累计
// 应收单（ApplyReceipt 的 SQL 条件防超收竞态）。累计失败仅记录
// （收款已批，经审计对账处理），不回滚审批。
func (s *ReceiptService) ApplyApproved(ctx context.Context, receiptID uint32) error {
	receipt, err := s.receiptRepo.Get(ctx, &financeV1.GetReceiptRequest{
		QueryBy: &financeV1.GetReceiptRequest_Id{Id: receiptID},
	})
	if err != nil {
		return err
	}
	if receipt.GetStatus() != financeV1.Receipt_PENDING {
		return financeV1.ErrorConflict("receipt is not pending")
	}

	if err := s.receiptRepo.TransitionStatus(ctx, receiptID,
		financeV1.Receipt_PENDING, financeV1.Receipt_APPLIED); err != nil {
		return err
	}

	if _, err := s.receivableRepo.ApplyReceipt(ctx, receipt.GetReceivableId(), receipt.GetAmount()); err != nil {
		s.log.Errorf("apply receivable for receipt %d failed after approval: %s", receiptID, err.Error())
	}

	// 总账过账：借 资金（现金/银行） / 贷 应收账款。软失败。
	s.journal.PostReceiptApplied(ctx, receiptID, receipt.GetAmount(),
		receipt.GetMethod() == financeV1.Receipt_CASH)

	return nil
}

// RejectApproved 审批拒绝：PENDING→REJECTED，不影响应收单。
func (s *ReceiptService) RejectApplied(ctx context.Context, receiptID uint32) error {
	return s.receiptRepo.TransitionStatus(ctx, receiptID,
		financeV1.Receipt_PENDING, financeV1.Receipt_REJECTED)
}
