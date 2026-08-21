package service

import (
	"context"
	"fmt"

	"go-wind-erp/app/core/service/internal/data"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
	internalMessageV1 "go-wind-erp/api/gen/go/internal_message/service/v1"
)

// approvalNotifier 审批事件站内信通知（审结时通知申请人）。
// 通知失败仅记录——通知是尽力而为的辅助通道，不阻塞审批主流程。
type approvalNotifier struct {
	messageRepo   *data.InternalMessageRepo
	recipientRepo *data.InternalMessageRecipientRepo
	log           interface {
		Errorf(format string, args ...any)
	}
}

// notifyResolved 审结通知：标题带结果与业务类型，收件人=申请人。
func (n *approvalNotifier) notifyResolved(ctx context.Context, approval *approvalV1.ApprovalRequest, approved bool) error {
	if n == nil || approval.GetApplicantId() == 0 {
		return nil
	}

	result := "已驳回"
	if approved {
		result = "已通过"
	}

	msg, err := n.messageRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title:   strPtr(fmt.Sprintf("审批%s：%s", result, approval.GetTitle())),
			Content: strPtr(fmt.Sprintf("你的审批单（%s / %s）%s。", approval.GetBizType(), approval.GetBizRef(), result)),
			Status:  internalMessageV1.InternalMessage_PUBLISHED.Enum(),
			Type:    internalMessageV1.InternalMessage_NOTIFICATION.Enum(),
		},
	})
	if err != nil {
		return err
	}

	_, err = n.recipientRepo.Create(ctx, &internalMessageV1.InternalMessageRecipient{
		MessageId:       msg.Id,
		RecipientUserId: approval.ApplicantId,
	})
	return err
}

// notifyReplenishmentDraft 补货审批通过后的下游事件通知：告知申请人草稿
// 采购单已自动创建、待完善提交。收件人=审批申请人。
func (n *approvalNotifier) notifyReplenishmentDraft(
	ctx context.Context,
	approval *approvalV1.ApprovalRequest,
	poNumber string,
) error {
	if n == nil || approval.GetApplicantId() == 0 {
		return nil
	}

	msg, err := n.messageRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title: strPtr(fmt.Sprintf("补货已受理：草稿采购单 %s 已创建", poNumber)),
			Content: strPtr(fmt.Sprintf(
				"你发起的补货建议（%s）已通过审批，系统已按最近供应商自动创建草稿采购单 %s，请完善单价后提交。",
				approval.GetBizRef(), poNumber,
			)),
			Status: internalMessageV1.InternalMessage_PUBLISHED.Enum(),
			Type:   internalMessageV1.InternalMessage_NOTIFICATION.Enum(),
		},
	})
	if err != nil {
		return err
	}

	_, err = n.recipientRepo.Create(ctx, &internalMessageV1.InternalMessageRecipient{
		MessageId:       msg.Id,
		RecipientUserId: approval.ApplicantId,
	})
	return err
}

// notifyPOAutoCompleted 全额收货自动完结的下游事件通知：告知采购单创建人
// 单据已收齐并自动完结。收件人=采购单创建者。
func (n *approvalNotifier) notifyPOAutoCompleted(
	ctx context.Context,
	poNumber string,
	creatorUserID uint32,
) error {
	if n == nil || creatorUserID == 0 {
		return nil
	}

	creator := creatorUserID
	msg, err := n.messageRepo.Create(ctx, &internalMessageV1.CreateInternalMessageRequest{
		Data: &internalMessageV1.InternalMessage{
			Title:   strPtr(fmt.Sprintf("采购单 %s 已全额收货并自动完结", poNumber)),
			Content: strPtr(fmt.Sprintf("采购单 %s 的全部明细已收货完成，单据已自动完结。", poNumber)),
			Status:  internalMessageV1.InternalMessage_PUBLISHED.Enum(),
			Type:    internalMessageV1.InternalMessage_NOTIFICATION.Enum(),
		},
	})
	if err != nil {
		return err
	}

	_, err = n.recipientRepo.Create(ctx, &internalMessageV1.InternalMessageRecipient{
		MessageId:       msg.Id,
		RecipientUserId: &creator,
	})
	return err
}
