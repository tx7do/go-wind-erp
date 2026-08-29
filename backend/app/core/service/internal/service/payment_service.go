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

// 付款审批联动约定：biz_type="PAYMENT"，biz_ref="PAYMENT:{id}"。
const (
	paymentApprovalBizType = "PAYMENT"
	paymentApprovalBizRef  = "PAYMENT:%d"
)

// PaymentService 付款服务（Create=提交付款申请，经审批后入账）
type PaymentService struct {
	financeV1.UnimplementedPaymentServiceServer

	log *log.Helper

	paymentRepo         *data.PaymentRepo
	payableRepo         *data.PayableRepo
	approvalRequestRepo *data.ApprovalRequestRepo
	journal             *journalPoster
}

func NewPaymentService(
	ctx *bootstrap.Context,
	paymentRepo *data.PaymentRepo,
	payableRepo *data.PayableRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
	journal *journalPoster,
) *PaymentService {
	svc := &PaymentService{
		log:                 ctx.NewLoggerHelper("payment/service/core-service"),
		paymentRepo:         paymentRepo,
		payableRepo:         payableRepo,
		approvalRequestRepo: approvalRequestRepo,
		journal:             journal,
	}

	return svc
}

func (s *PaymentService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPaymentResponse, error) {
	return s.paymentRepo.List(ctx, req)
}

func (s *PaymentService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.CountPaymentResponse, error) {
	count, err := s.paymentRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &financeV1.CountPaymentResponse{Count: uint64(count)}, nil
}

func (s *PaymentService) Get(ctx context.Context, req *financeV1.GetPaymentRequest) (*financeV1.Payment, error) {
	return s.paymentRepo.Get(ctx, req)
}

// Create 提交付款申请：落 PENDING 付款记录并自动生成审批单
// （biz_type="PAYMENT"，biz_ref="PAYMENT:{id}"）。审批通过后经
// [ApplyApproved] 真正入账（累计应付单）；驳回则 [RejectApplied]。
// 预校验仅做只读检查（防呆）；权威的防超付/状态守卫在入账时原子执行。
func (s *PaymentService) Create(ctx context.Context, req *financeV1.CreatePaymentRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}
	if req.Data.GetPayableId() == 0 {
		return nil, financeV1.ErrorBadRequest("payable_id is required")
	}
	amount := req.Data.GetAmount()
	if amount <= 0 {
		return nil, financeV1.ErrorBadRequest("amount must be positive")
	}

	payable, err := s.payableRepo.Get(ctx, &financeV1.GetPayableRequest{
		QueryBy: &financeV1.GetPayableRequest_Id{Id: req.Data.GetPayableId()},
	})
	if err != nil {
		return nil, err
	}
	if _, overflow := addChecked(payable.GetPaidAmount(), amount); overflow {
		return nil, financeV1.ErrorBadRequest("paid amount overflow")
	}
	// 只读预检：明显超付直接拒绝（竞态仍由入账时的 SQL 条件兜底）。
	if payable.GetPaidAmount()+amount > payable.GetAmount() {
		return nil, financeV1.ErrorBadRequest("payment exceeds payable amount")
	}

	payment, err := s.paymentRepo.Create(ctx, &financeV1.CreatePaymentRequest{
		Data: &financeV1.Payment{
			PayableId: req.Data.PayableId,
			Amount:    req.Data.Amount,
			Method:    req.Data.Method,
			Remark:    req.Data.Remark,
		},
	})
	if err != nil {
		return nil, err
	}

	// 生成付款审批单；失败不影响付款记录（PENDING 可重提审批）。
	if _, aerr := s.approvalRequestRepo.Create(ctx, &approvalV1.CreateApprovalRequestRequest{
		Data: &approvalV1.ApprovalRequest{
			Title:    trans.Ptr(fmt.Sprintf("付款申请 %s（%d 分 → 应付 %s）", payment.GetPaymentNumber(), amount, payable.GetPayableNumber())),
			BizType:  trans.Ptr(paymentApprovalBizType),
			BizRef:   trans.Ptr(fmt.Sprintf(paymentApprovalBizRef, payment.GetId())),
			Summary:  trans.Ptr(fmt.Sprintf("供应商 %s；应付 %d 分，已付 %d 分", payable.GetSupplierCode(), payable.GetAmount(), payable.GetPaidAmount())),
		},
	}); aerr != nil {
		s.log.Errorf("create approval for payment %d failed: %s", payment.GetId(), aerr.Error())
	}

	return &emptypb.Empty{}, nil
}

// ApplyApproved 审批通过后入账：PENDING→APPLIED（条件迁移）后原子累计
// 应付单（ApplyPayment 的 SQL 条件防超付竞态）。累计失败仅记录
// （付款已批，经审计对账处理），不回滚审批。
func (s *PaymentService) ApplyApproved(ctx context.Context, paymentID uint32) error {
	payment, err := s.paymentRepo.Get(ctx, &financeV1.GetPaymentRequest{
		QueryBy: &financeV1.GetPaymentRequest_Id{Id: paymentID},
	})
	if err != nil {
		return err
	}
	if payment.GetStatus() != financeV1.Payment_PENDING {
		return financeV1.ErrorConflict("payment is not pending")
	}

	if err := s.paymentRepo.TransitionStatus(ctx, paymentID,
		financeV1.Payment_PENDING, financeV1.Payment_APPLIED); err != nil {
		return err
	}

	if _, err := s.payableRepo.ApplyPayment(ctx, payment.GetPayableId(), payment.GetAmount()); err != nil {
		s.log.Errorf("apply payable for payment %d failed after approval: %s", paymentID, err.Error())
	}

	// 总账过账：借 应付账款 / 贷 资金（现金/银行）。软失败。
	s.journal.PostPaymentApplied(ctx, paymentID, payment.GetAmount(),
		payment.GetMethod() == financeV1.Payment_CASH)

	return nil
}

// RejectApproved 审批拒绝：PENDING→REJECTED，不影响应付单。
func (s *PaymentService) RejectApplied(ctx context.Context, paymentID uint32) error {
	return s.paymentRepo.TransitionStatus(ctx, paymentID,
		financeV1.Payment_PENDING, financeV1.Payment_REJECTED)
}
