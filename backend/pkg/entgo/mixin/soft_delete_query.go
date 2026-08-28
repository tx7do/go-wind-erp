package mixin

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/mixin"

	"github.com/tx7do/go-crud/entgo/interceptor"
)

// SoftDeleteQuery 为实体注册软删除查询拦截器（注入 deleted_at IS NULL）。
//
// 仅注册拦截器、不定义字段——deleted_at 已由 TimeAt mixin 提供，避免重复列。
// 查询级拦截（不含变更钩子）：本项目的删除均为硬删除，软删除行只可能来自
// 手工 SQL 或历史数据，查询级过滤即可保证它们不再出现在业务读取中。
//
// 仅可添加到含 TimeAt（即含 deleted_at 列）的实体；无该列的表（审计日志、
// 订单明细、move_line）不得使用，否则注入的谓词会导致 SQL 报错。
type SoftDeleteQuery struct {
	mixin.Schema
}

func (SoftDeleteQuery) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{interceptor.SoftDeleteInterceptor()}
}
