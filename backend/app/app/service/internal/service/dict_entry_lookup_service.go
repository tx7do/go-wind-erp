package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	dictV1 "go-wind-erp/api/gen/go/dict/service/v1"
)

// DictEntryLookup 是 app 侧字典只读查询 facade：仅暴露 ListByTypeCode，
// 将请求委派给 core 的 DictEntryService gRPC 客户端。不持有写操作方法，
// 因此即便 core 客户端支持 Create/Update/Delete，本 facade 也不对外提供。
type DictEntryLookup struct {
	appV1.DictEntryLookupHTTPServer

	dictEntryServiceClient dictV1.DictEntryServiceClient

	log *log.Helper
}

func NewDictEntryLookup(
	ctx *bootstrap.Context,
	dictEntryServiceClient dictV1.DictEntryServiceClient,
) *DictEntryLookup {
	return &DictEntryLookup{
		dictEntryServiceClient: dictEntryServiceClient,
		log:                    ctx.NewLoggerHelper("dict/service/app-service"),
	}
}

func (s *DictEntryLookup) ListByTypeCode(ctx context.Context, req *dictV1.ListDictEntryByTypeCodeRequest) (*dictV1.ListDictEntryByTypeCodeResponse, error) {
	return s.dictEntryServiceClient.ListByTypeCode(ctx, req)
}
