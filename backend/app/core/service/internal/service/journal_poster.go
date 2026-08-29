package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-erp/app/core/service/internal/data"
	"go-wind-erp/app/core/service/internal/data/ent"

	"go-wind-erp/pkg/constants"
)

// journalPoster 业务事件过账器（简易总账）：把收发货/审批/收付款翻译成
// 平衡分录。拣货路径的过账与业务回写同事务（账实一致优先，失败即回滚）；
// 审批/收付款路径无外层事务，独立事务过账、失败仅记录（镜像应付单的
// 软失败模式——业务事实优先，凭证可对账修复）。
type journalPoster struct {
	journalRepo *data.JournalRepo
	log         *log.Helper
}

func newJournalPoster(journalRepo *data.JournalRepo, log *log.Helper) *journalPoster {
	return &journalPoster{journalRepo: journalRepo, log: log}
}

// PostReceivingTx 采购入库：借 库存商品 / 贷 应付账款（数量×采购单价）。
func (p *journalPoster) PostReceivingTx(
	ctx context.Context, tx *ent.Tx, pickingID uint32, skuCode string, qty int64, unitCost int64,
) error {
	amount := qty * unitCost
	if amount == 0 {
		return nil
	}
	return p.journalRepo.PostTx(ctx, tx,
		fmt.Sprintf("采购入库 %s ×%d", skuCode, qty),
		fmt.Sprintf("STOCK_PICKING:%d", pickingID),
		[]data.JournalLineInput{
			{AccountCode: constants.AccountCodeInventory, Summary: "入库", Debit: amount},
			{AccountCode: constants.AccountCodeAP, Summary: "应付", Credit: amount},
		})
}

// PostDeliveryTx 销售出库（COGS）：借 主营业务成本 / 贷 库存商品
// （数量×冻结加权成本）。
func (p *journalPoster) PostDeliveryTx(
	ctx context.Context, tx *ent.Tx, pickingID uint32, skuCode string, qty, unitCost int64,
) error {
	amount := qty * unitCost
	if amount == 0 {
		return nil
	}
	return p.journalRepo.PostTx(ctx, tx,
		fmt.Sprintf("销售出库 %s ×%d", skuCode, qty),
		fmt.Sprintf("STOCK_PICKING:%d", pickingID),
		[]data.JournalLineInput{
			{AccountCode: constants.AccountCodeCOGS, Summary: "销售成本", Debit: amount},
			{AccountCode: constants.AccountCodeInventory, Summary: "出库", Credit: amount},
		})
}

// PostSalesReturnStockTx 销退库存腿：借 库存 / 贷 成本（按回货均价）。
func (p *journalPoster) PostSalesReturnStockTx(
	ctx context.Context, tx *ent.Tx, pickingID uint32, skuCode string, qty, avgCost int64,
) error {
	amount := qty * avgCost
	if amount == 0 {
		return nil
	}
	return p.journalRepo.PostTx(ctx, tx,
		fmt.Sprintf("销退成本冲回 %s ×%d", skuCode, qty),
		fmt.Sprintf("STOCK_PICKING:%d", pickingID),
		[]data.JournalLineInput{
			{AccountCode: constants.AccountCodeInventory, Summary: "退货入库", Debit: amount},
			{AccountCode: constants.AccountCodeCOGS, Summary: "成本冲回", Credit: amount},
		})
}

// PostSalesReturnRevenueTx 销退收入腿：借 收入 / 贷 应收（数量×销售单价）。
// 与库存腿分开过账（收入腿在 so_item 回写分支，能取到明细单价）。
func (p *journalPoster) PostSalesReturnRevenueTx(
	ctx context.Context, tx *ent.Tx, pickingID uint32, skuCode string, qty, salePrice int64,
) error {
	amount := qty * salePrice
	if amount == 0 {
		return nil
	}
	return p.journalRepo.PostTx(ctx, tx,
		fmt.Sprintf("销退收入冲回 %s ×%d", skuCode, qty),
		fmt.Sprintf("STOCK_PICKING:%d", pickingID),
		[]data.JournalLineInput{
			{AccountCode: constants.AccountCodeRevenue, Summary: "收入冲回", Debit: amount},
			{AccountCode: constants.AccountCodeAR, Summary: "应收冲回", Credit: amount},
		})
}

// PostPurchaseReturnTx 采购退货：借 应付账款 / 贷 库存商品。
func (p *journalPoster) PostPurchaseReturnTx(
	ctx context.Context, tx *ent.Tx, pickingID uint32, skuCode string, qty, avgCost int64,
) error {
	amount := qty * avgCost
	if amount == 0 {
		return nil
	}
	return p.journalRepo.PostTx(ctx, tx,
		fmt.Sprintf("采购退货 %s ×%d", skuCode, qty),
		fmt.Sprintf("STOCK_PICKING:%d", pickingID),
		[]data.JournalLineInput{
			{AccountCode: constants.AccountCodeAP, Summary: "应付冲回", Debit: amount},
			{AccountCode: constants.AccountCodeInventory, Summary: "退货出库", Credit: amount},
		})
}

// PostStockGainLossTx 盘点损益：盘盈 借 库存 / 贷 待处理财产损溢；
// 盘亏反向。
func (p *journalPoster) PostStockGainLossTx(
	ctx context.Context, tx *ent.Tx, pickingID uint32, skuCode string, gain bool, qty, avgCost int64,
) error {
	amount := qty * avgCost
	if amount == 0 {
		return nil
	}
	lines := []data.JournalLineInput{
		{AccountCode: constants.AccountCodePendingLoss, Summary: "盘亏", Debit: amount},
		{AccountCode: constants.AccountCodeInventory, Summary: "盘亏出库", Credit: amount},
	}
	if gain {
		lines = []data.JournalLineInput{
			{AccountCode: constants.AccountCodeInventory, Summary: "盘盈入库", Debit: amount},
			{AccountCode: constants.AccountCodePendingLoss, Summary: "盘盈", Credit: amount},
		}
	}
	return p.journalRepo.PostTx(ctx, tx,
		fmt.Sprintf("盘点%s %s ×%d", map[bool]string{true: "盘盈", false: "盘亏"}[gain], skuCode, qty),
		fmt.Sprintf("STOCK_PICKING:%d", pickingID),
		lines)
}

// PostRevenueRecognition 销售单获批（收入确认）：借 应收账款 / 贷 主营业务收入。
// 独立事务软失败（与应收单生成同模式）。
func (p *journalPoster) PostRevenueRecognition(ctx context.Context, soID uint32, soNumber string, totalAmount int64) {
	if totalAmount == 0 {
		return
	}
	if err := p.journalRepo.Post(ctx,
		fmt.Sprintf("销售确认收入 %s", soNumber),
		fmt.Sprintf("SALES_ORDER:%d", soID),
		[]data.JournalLineInput{
			{AccountCode: constants.AccountCodeAR, Summary: "应收", Debit: totalAmount},
			{AccountCode: constants.AccountCodeRevenue, Summary: "收入", Credit: totalAmount},
		}); err != nil {
		p.log.Errorf("post revenue recognition for so %d failed: %s", soID, err.Error())
	}
}

// PostReceiptApplied 收款入账：借 资金（现金/银行） / 贷 应收账款。
func (p *journalPoster) PostReceiptApplied(ctx context.Context, receiptID uint32, amount int64, cash bool) {
	if amount == 0 {
		return
	}
	fund := constants.AccountCodeBank
	fundSummary := "银行存款"
	if cash {
		fund = constants.AccountCodeCash
		fundSummary = "库存现金"
	}
	if err := p.journalRepo.Post(ctx,
		"收款入账",
		fmt.Sprintf("RECEIPT:%d", receiptID),
		[]data.JournalLineInput{
			{AccountCode: fund, Summary: fundSummary, Debit: amount},
			{AccountCode: constants.AccountCodeAR, Summary: "应收核销", Credit: amount},
		}); err != nil {
		p.log.Errorf("post receipt %d failed: %s", receiptID, err.Error())
	}
}

// PostPaymentApplied 付款入账：借 应付账款 / 贷 资金。
func (p *journalPoster) PostPaymentApplied(ctx context.Context, paymentID uint32, amount int64, cash bool) {
	if amount == 0 {
		return
	}
	fund := constants.AccountCodeBank
	fundSummary := "银行存款"
	if cash {
		fund = constants.AccountCodeCash
		fundSummary = "库存现金"
	}
	if err := p.journalRepo.Post(ctx,
		"付款入账",
		fmt.Sprintf("PAYMENT:%d", paymentID),
		[]data.JournalLineInput{
			{AccountCode: constants.AccountCodeAP, Summary: "应付核销", Debit: amount},
			{AccountCode: fund, Summary: fundSummary, Credit: amount},
		}); err != nil {
		p.log.Errorf("post payment %d failed: %s", paymentID, err.Error())
	}
}

// NewJournalPoster wire 提供者。
func NewJournalPoster(ctx *bootstrap.Context, journalRepo *data.JournalRepo) *journalPoster {
	return newJournalPoster(journalRepo, ctx.NewLoggerHelper("journal/poster/core-service"))
}
