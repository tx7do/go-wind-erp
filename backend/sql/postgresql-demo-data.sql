BEGIN;

SET LOCAL search_path = public, pg_catalog;

-- 一次性清理相关表并重置自增（包含外键依赖）
TRUNCATE TABLE public.sys_org_units,
               public.sys_positions,
               public.sys_tasks,
               public.sys_login_policies,
               public.sys_dict_types,
               public.sys_dict_entries,
               public.internal_message_categories,
               public.inv_warehouses,
               public.inv_stock_locations,
               public.inv_stock_quants,
               public.inv_stock_pickings,
               public.inv_stock_moves,
               public.inv_stock_move_lines,
               public.prd_products,
               public.pur_suppliers,
               public.pur_purchase_orders,
               public.pur_purchase_order_items,
               public.fin_payables,
               public.fin_payments,
               public.apr_approval_requests
RESTART IDENTITY CASCADE;

-- ----------------------------
-- 插入 sys_tenants 租户
-- ----------------------------
INSERT INTO public.sys_tenants(id, name, code, type, audit_status, status, admin_user_id, created_at)
VALUES (1, '测试租户', 'super', 'PAID', 'APPROVED', 'ON', 2, now())
;
SELECT setval('sys_tenants_id_seq', (SELECT MAX(id) FROM sys_tenants));

-- ----------------------------
-- 插入 sys_users 租户管理员用户
-- ----------------------------
INSERT INTO public.sys_users (id, tenant_id, username, nickname, realname, email, gender, created_at)
VALUES
    -- 2. 租户管理员（TENANT_ADMIN）
    (2, 1, 'tenant_admin', '租户管理', '张管理员', 'tenant@company.com', 'MALE', now())
;
SELECT setval('sys_users_id_seq', (SELECT MAX(id) FROM sys_users));

-- ----------------------------
-- 插入 sys_user_credentials 用户凭证（密码统一为admin，哈希值与原admin一致，方便测试）
-- ----------------------------
INSERT INTO public.sys_user_credentials (tenant_id, user_id, identity_type, identifier, credential_type, credential, status,
                                         is_primary, created_at)
VALUES
    -- 租户管理员（对应users表id=2，tenant_id=1）
    (1, 2, 'USERNAME', 'tenant_admin', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', true, now()),
    (1, 2, 'EMAIL', 'tenant@company.com', 'PASSWORD_HASH', '$2a$10$yajZDX20Y40FkG0Bu4N19eXNqRizez/S9fK63.JxGkfLq.RoNKR/a', 'ENABLED', false, now())
;
SELECT setval('sys_user_credentials_id_seq', (SELECT MAX(id) FROM sys_user_credentials));

-- ----------------------------
-- 插入租户管理员角色（从模板克隆到租户，对齐 CreateTenantWithAdminUser 流程）
--
-- 模板角色（code='template:tenant:manager'，tenant_id=0）由服务启动时的系统初始化创建。
-- ent privacy 租户过滤要求权限记录必须带租户维度：租户管理员必须持有克隆出的
-- 租户角色（tenant_id=1）及其权限记录，直接绑定模板角色会因 tenant_id 不匹配
-- 查不到任何权限码（accessCodes 为空 → 前端路由全部被过滤 → 无菜单）。
-- 角色按 code 查找引用，权限按 code 关联，不依赖系统初始化的自增 ID。
-- ----------------------------
INSERT INTO public.sys_roles (tenant_id, name, code, type, is_protected, sort_order, status, description, created_at)
SELECT 1, '租户管理员', 'tenant:manager', 'TENANT', true, 2, 'ON',
       '租户管理员角色，拥有租户内所有功能的操作权限', now()
WHERE NOT EXISTS (SELECT 1 FROM public.sys_roles WHERE tenant_id = 1 AND code = 'tenant:manager');
SELECT setval('sys_roles_id_seq', (SELECT MAX(id) FROM sys_roles));

INSERT INTO public.sys_role_metadata (tenant_id, role_id, is_template, template_version, sync_policy, scope, custom_overrides, created_at)
SELECT 1, r.id, false, 1, 'AUTO', 'TENANT', '{}'::jsonb, now()
FROM public.sys_roles r
WHERE r.tenant_id = 1 AND r.code = 'tenant:manager'
  AND NOT EXISTS (SELECT 1 FROM public.sys_role_metadata m WHERE m.role_id = r.id);
SELECT setval('sys_role_metadata_id_seq', (SELECT MAX(id) FROM sys_role_metadata));

INSERT INTO public.sys_role_permissions (tenant_id, role_id, permission_id, effect, priority, status, created_at)
SELECT 1, r.id, p.id, 'ALLOW', 0, 'ON', now()
FROM public.sys_roles r
JOIN public.sys_permissions p ON p.code IN ('sys:access_backend', 'sys:tenant_manager')
WHERE r.tenant_id = 1 AND r.code = 'tenant:manager'
  AND NOT EXISTS (
      SELECT 1 FROM public.sys_role_permissions rp
      WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
SELECT setval('sys_role_permissions_id_seq', (SELECT MAX(id) FROM sys_role_permissions));

INSERT INTO public.sys_user_roles (tenant_id, user_id, role_id, is_primary, status, assigned_at, created_at)
SELECT 1, 2, r.id, true, 'ACTIVE', now(), now()
FROM public.sys_roles r
WHERE r.tenant_id = 1 AND r.code = 'tenant:manager'
  AND NOT EXISTS (
      SELECT 1 FROM public.sys_user_roles ur
      WHERE ur.user_id = 2 AND ur.role_id = r.id
  );
SELECT setval('sys_user_roles_id_seq', (SELECT MAX(id) FROM sys_user_roles));

-- ----------------------------
-- 插入 sys_org_units 组织架构单元
-- ----------------------------
INSERT INTO public.sys_org_units (id, tenant_id, parent_id, type, name, code, description, path, sort_order, leader_id, status, created_at)
VALUES
    (1, 1, NULL, 'COMPANY', 'XX集团总部', 'HEADQUARTERS', '集团核心管理机构，统筹全集团战略规划、业务管控及资源调配', '/1', 1, 1, 'ON', now()),
    (2, 1, 1, 'DIVISION', '技术部', 'TECH', '负责集团整体技术架构规划、研发管理、系统运维及技术创新', '/1/2', 2, 5, 'ON', now()),
    (3, 1, 1, 'DIVISION', '财务部', 'FIN', '负责集团财务核算、资金管理、税务筹划、预算编制及财务风控', '/1/3', 3, 8, 'ON', now()),
    (4, 1, 1, 'DIVISION', '人事部', 'HR', '负责人力资源规划、招聘配置、薪酬绩效、员工培训及组织发展', '/1/4', 4, 9, 'ON', now()),
    (5, 1, 2, 'DEPARTMENT', '研发一部', 'DEV-1', '聚焦新能源领域产品研发、技术迭代及核心模块开发', '/1/2/5', 1, 6, 'ON', now()),
    (6, 1, 1, 'REGION', '华北大区', 'NORTH', '负责华北区域市场运营、客户维护、销售管理及本地化服务落地', '/1/6', 3, 12, 'ON', now()),
    (7, 1, 1, 'SUBSIDIARY', '广州分公司', 'GZ', '负责华南区域（广州及周边）业务拓展、客户服务及本地化运营', '/1/7', 5, 2, 'ON', now()),
    (8, 1, 1, 'SUBSIDIARY', '深圳子公司', 'SZ', '负责深圳区域市场开拓、科技创新业务落地及高端客户对接', '/1/8', 6, 4, 'ON', now()),
    (9, 1, 1, 'DIVISION', '销售部', 'SALES', '统筹集团整体销售策略制定、销售团队管理及业绩目标达成', '/1/9', 7, 16, 'ON', now()),
    (10, 1, 9, 'DEPARTMENT', '海外事业部', 'INTL', '负责海外市场拓展、国际客户合作、跨境业务管理及本地化运营', '/1/9/10', 1, 17, 'ON', now()),
    (11, 1, 10, 'TEAM', '海外销售组', 'INTL-SALES-1', '具体执行海外市场销售任务，跟进客户需求及订单落地', '/1/9/10/11', 1, 18, 'ON', now()),
    (12, 1, 5, 'PROJECT', '新能源项目组', 'NEO-PROJ', '专项负责新能源项目的研发、落地、运营及成果转化', '/1/2/5/12', 1, 6, 'ON', now()),
    (13, 1, 1, 'COMMITTEE', '审计委员会', 'AUDIT', '独立开展集团内部审计、风控检查、合规监督及问题整改跟进', '/1/13', 8, 12, 'ON', now()),
    (14, 1, 1, 'DEPARTMENT', '客服部', 'CS', '负责全集团客户咨询、投诉处理、售后服务及客户满意度提升', '/1/14', 9, 11, 'ON', now()),
    (15, 1, 14, 'TEAM', '客服一组', 'CS-1', '承接华南区域客户服务、售后问题处理及客户关系维护', '/1/14/15', 1, 20, 'ON', now())
;
SELECT setval('sys_org_units_id_seq', (SELECT MAX(id) FROM sys_org_units));

-- ----------------------------
-- 插入 sys_positions 岗位数据
-- ----------------------------
INSERT INTO public.sys_positions (id, tenant_id, type, name, code, org_unit_id, reports_to_position_id, description, job_family, job_grade, level, headcount, is_key_position, status, sort_order, created_at)
VALUES
    (1, 1, 'LEADER', '技术总监', 'TECH-DIRECTOR-001', 2, NULL, '负责公司整体技术战略规划、团队管理及核心技术决策', 'TECH', 1, 1, 1, true, 'ON', 1, now()),
    (2, 1, 'MANAGER', '技术部经理', 'TECH-MANAGER-001', 2, 1, '负责技术部日常管理、项目排期及团队协作', 'TECH', 2, 2, 1, true, 'ON', 2, now()),
    (3, 1, 'MANAGER', '前端主管', 'TECH-FE-LEADER-001', 2, 2, '负责前端团队开发管理、技术方案评审及需求落地', 'TECH', 3, 3, 3, false, 'ON', 3, now()),
    (4, 1, 'MANAGER', '后端主管', 'TECH-BE-LEADER-001', 2, 2, '负责后端服务架构设计、数据库优化及接口开发管理', 'TECH', 4, 3, 3, false, 'ON', 4, now()),
    (5, 1, 'REGULAR', '前端开发专员', 'TECH-FE-DEV-001', 2, 3, '负责Web/移动端前端页面开发、交互实现及兼容性优化', 'TECH', 5, 4, 5, false, 'ON', 5, now()),
    (6, 1, 'REGULAR', '后端开发专员', 'TECH-BE-DEV-001', 2, 4, '负责后端接口开发、业务逻辑实现及系统稳定性维护', 'TECH', 6, 4, 5, false, 'ON', 6, now()),
    (7, 1, 'REGULAR', '测试工程师', 'TECH-TEST-001', 2, 2, '负责项目功能测试、性能测试及自动化测试脚本开发', 'TECH', 3, 4, 3, false, 'ON', 7, now()),
    (8, 1, 'LEADER', '人力总监', 'HR-DIRECTOR-001', 2, NULL, '负责人力资源战略规划、组织架构设计及人才梯队建设', 'HR', 1, 1, 1, true, 'ON', 1, now()),
    (9, 1, 'MANAGER', '招聘主管', 'HR-RECRUIT-LEADER-001', 2, 8, '负责公司各部门招聘需求对接、简历筛选及面试安排', 'HR', 2, 2, 1, false, 'ON', 2, now()),
    (10, 1, 'REGULAR', '薪酬绩效专员', 'HR-C&P-001', 2, 8, '负责员工薪酬核算、绩效考核制度落地及社保公积金管理', 'HR', 3, 2, 1, false, 'ON', 3, now()),
    (11, 1, 'REGULAR', 'HRBP', 'HR-BP-001', 2, 8, '对接业务部门，提供人力资源支持（入离职、员工关系等）', 'HR', 4, 2, 1, false, 'ON', 4, now()),
    (12, 1, 'LEADER', '财务总监', 'FIN-DIRECTOR-001', 2, NULL, '负责公司财务战略、预算管理及财务风险控制', 'FIN', 1, 1, 1, true, 'ON', 1, now()),
    (13, 1, 'MANAGER', '会计主管', 'FIN-ACCOUNT-LEADER-001', 2, 12, '负责账务处理、财务报表编制及税务申报管理', 'FIN', 2, 2, 1, false, 'ON', 2, now()),
    (14, 1, 'REGULAR', '出纳专员', 'FIN-CASHIER-001', 2, 13, '负责日常资金收付、银行对账及票据管理', 'FIN', 3, 3, 1, false, 'ON', 3, now()),
    (15, 1, 'REGULAR', '成本会计', 'FIN-COST-001', 2, 13, '负责成本核算、成本分析及成本控制方案制定', 'FIN', 4, 3, 1, false, 'ON', 4, now()),
    (16, 1, 'LEADER', '市场总监', 'MKT-DIRECTOR-001', 4, NULL, '负责市场战略规划、品牌建设及营销活动策划', 'MKT', 1, 1, 1, true, 'ON', 1, now()),
    (17, 1, 'MANAGER', '新媒体运营主管', 'MKT-NEWS-LEADER-001', 4, 16, '负责新媒体平台内容运营及用户增长', 'MKT', 2, 2, 1, false, 'ON', 2, now()),
    (18, 1, 'REGULAR', '活动策划专员', 'MKT-EVENT-001', 4, 16, '负责线下活动策划、执行及效果复盘', 'MKT', 3, 3, 1, false, 'ON', 3, now()),
    (19, 1, 'REGULAR', '市场调研专员', 'MKT-RESEARCH-001', 4, 16, '负责行业动态调研、竞品分析及市场趋势报告撰写', 'MKT', 4, 3, 1, false, 'ON', 4, now()),
    (20, 1, 'REGULAR', '行政助理', 'ADMIN-ASSIST-001', 2, 8, '负责办公用品采购、会议安排等行政工作（已合并至HRBP）', 'ADMIN', 5, 5, 1, false, 'OFF', 5, now())
;
SELECT setval('sys_positions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM sys_positions));

-- ----------------------------
-- 插入 sys_tasks 调度任务
-- ----------------------------
INSERT INTO public.sys_tasks(type, type_name, task_payload, cron_spec, enable, created_at)
VALUES
    ('PERIODIC', 'backup', '{ "name": "test"}', '0 * * * *', true, now())
;
SELECT setval('sys_tasks_id_seq', (SELECT MAX(id) FROM sys_tasks));

-- ----------------------------
-- 插入 sys_login_policies 登录策略
-- ----------------------------
INSERT INTO public.sys_login_policies(id, target_id, type, method, value, reason, created_at)
VALUES
    (1, 1, 'BLACKLIST', 'IP', '127.0.0.1', '无理由', now()),
    (2, 1, 'WHITELIST', 'MAC', '00:1B:44:11:3A:B7 ', '无理由', now())
;
SELECT setval('sys_login_policies_id_seq', (SELECT MAX(id) FROM sys_login_policies));

-- ----------------------------
-- 插入 sys_dict_types 字典类型
-- ----------------------------
INSERT INTO public.sys_dict_types (
    id, type_code, type_name, sort_order, is_enabled, created_at, updated_at
) VALUES
      (1, 'USER_STATUS', '用户状态', 10, true, now(), now()),
      (2, 'DEVICE_TYPE', '设备类型', 20, true, now(), now()),
      (3, 'ORDER_STATUS', '订单状态', 30, true, now(), now()),
      (4, 'GENDER', '性别', 40, true, now(), now()),
      (5, 'PAYMENT_METHOD', '支付方式', 50, true, now(), now()),
      (6, 'APPROVAL_BIZ_TYPE', '审批业务类型', 60, true, now(), now())
;
SELECT setval('sys_dict_types_id_seq', (SELECT MAX(id) FROM sys_dict_types));

-- ----------------------------
-- 插入 sys_dict_entries 字典条目
-- ----------------------------
INSERT INTO public.sys_dict_entries (
    id, type_id, entry_value, numeric_value, sort_order, is_enabled, created_at, updated_at, tenant_id
) VALUES
      -- 用户状态
      (1, 1, 'NORMAL', 1, 1, true, now(), now(), 0),
      (2, 1, 'FROZEN', 2, 2, true, now(), now(), 0),
      (3, 1, 'CANCELED', 3, 3, true, now(), now(), 0),
      -- 设备类型
      (4, 2, 'TEMP_SENSOR', 101, 1, true, now(), now(), 0),
      (5, 2, 'CURRENT_METER', 102, 2, true, now(), now(), 0),
      (6, 2, 'GAS_DETECTOR', 103, 3, false, now(), now(), 0),
      -- 订单状态
      (7, 3, 'PENDING', 1, 1, true, now(), now(), 0),
      (8, 3, 'PAID', 2, 2, true, now(), now(), 0),
      (9, 3, 'SHIPPED', 3, 3, true, now(), now(), 0),
      (10, 3, 'COMPLETED', 4, 4, true, now(), now(), 0),
      (11, 3, 'CANCELED', 5, 5, true, now(), now(), 0),
      -- 性别
      (12, 4, 'MALE', 1, 1, true, now(), now(), 0),
      (13, 4, 'FEMALE', 2, 2, true, now(), now(), 0),
      (14, 4, 'UNKNOWN', 0, 3, true, now(), now(), 0),
      -- 支付方式
      (15, 5, 'ALIPAY', 1, 1, true, now(), now(), 0),
      (16, 5, 'WECHAT', 2, 2, true, now(), now(), 0),
      (17, 5, 'UNIONPAY', 3, 3, true, now(), now(), 0),
      (18, 5, 'CASH', 4, 4, false, now(), now(), 0),
      -- 审批业务类型（审批单 biz_type 的展示标签来源）
      (19, 6, 'PURCHASE_ORDER', 0, 1, true, now(), now(), 0),
      (20, 6, 'REPLENISHMENT', 0, 2, true, now(), now(), 0),
      (21, 6, 'PAYMENT', 0, 3, true, now(), now(), 0)
;
SELECT setval('sys_dict_entries_id_seq', (SELECT MAX(id) FROM sys_dict_entries));

-- ----------------------------
-- 插入 sys_dict_entry_i18n 字典条目国际化
-- ----------------------------
INSERT INTO public.sys_dict_entry_i18n (
    entry_id, language_code, entry_label, description, sort_order, tenant_id, created_at, updated_at
) VALUES
      -- 用户状态
      (1, 'zh-CN', '正常', '用户可正常登录和操作', 1, 0, now(), now()),
      (2, 'zh-CN', '冻结', '因违规被临时冻结，需管理员解冻', 2, 0, now(), now()),
      (3, 'zh-CN', '注销', '用户主动注销，数据保留但不可登录', 3, 0, now(), now()),
      -- 设备类型
      (4, 'zh-CN', '温湿度传感器', '支持温度（-20~80℃）和湿度（0~100%RH）采集', 1, 0, now(), now()),
      (5, 'zh-CN', '电流仪表', '交流/直流电流测量，精度0.5级', 2, 0, now(), now()),
      (6, 'zh-CN', '气体探测器', '暂不支持，待硬件适配（2025Q4计划启用）', 3, 0, now(), now()),
      -- 订单状态
      (7, 'zh-CN', '待支付', '下单后未支付，超时自动取消', 1, 0, now(), now()),
      (8, 'zh-CN', '已支付', '支付成功，等待发货', 2, 0, now(), now()),
      (9, 'zh-CN', '已发货', '商品已出库，物流配送中', 3, 0, now(), now()),
      (10, 'zh-CN', '已完成', '用户确认收货，订单结束', 4, 0, now(), now()),
      (11, 'zh-CN', '已取消', '用户或系统取消订单', 5, 0, now(), now()),
      -- 性别
      (12, 'zh-CN', '男', '', 1, 0, now(), now()),
      (13, 'zh-CN', '女', '', 2, 0, now(), now()),
      (14, 'zh-CN', '未知', '用户未填写时默认值', 3, 0, now(), now()),
      -- 支付方式
      (15, 'zh-CN', '支付宝', '支持花呗、余额宝', 1, 0, now(), now()),
      (16, 'zh-CN', '微信支付', '需绑定微信', 2, 0, now(), now()),
      (17, 'zh-CN', '银联支付', '支持信用卡、储蓄卡', 3, 0, now(), now()),
      (18, 'zh-CN', '现金支付', '线下支付，已废弃（2025-01停用）', 4, 0, now(), now()),
      -- 审批业务类型
      (19, 'zh-CN', '采购订单', '采购单提交的审批请求', 1, 0, now(), now()),
      (20, 'zh-CN', '补货', '库存补货触发的审批请求', 2, 0, now(), now()),
      (21, 'zh-CN', '付款', '财务付款触发的审批请求', 3, 0, now(), now()),

      -- User Status
      (1, 'en-US', 'Normal', 'User can log in and operate normally', 1, 0, now(), now()),
      (2, 'en-US', 'Frozen', 'Temporarily frozen due to violation; requires admin to unfreeze', 2, 0, now(), now()),
      (3, 'en-US', 'Canceled', 'User voluntarily canceled; data retained but login disabled', 3, 0, now(), now()),

      -- Device Type
      (4, 'en-US', 'Temperature & Humidity Sensor', 'Supports temperature (-20~80°C) and humidity (0~100% RH) measurement', 1, 0, now(), now()),
      (5, 'en-US', 'Current Meter', 'Measures AC/DC current with 0.5-class accuracy', 2, 0, now(), now()),
      (6, 'en-US', 'Gas Detector', 'Not supported yet; hardware integration planned for Q4 2025', 3, 0, now(), now()),

      -- Order Status
      (7, 'en-US', 'Pending Payment', 'Order placed but not paid; auto-canceled if timeout', 1, 0, now(), now()),
      (8, 'en-US', 'Paid', 'Payment successful; awaiting shipment', 2, 0, now(), now()),
      (9, 'en-US', 'Shipped', 'Item has left warehouse; in transit', 3, 0, now(), now()),
      (10, 'en-US', 'Completed', 'User confirmed receipt; order closed', 4, 0, now(), now()),
      (11, 'en-US', 'Canceled', 'Order canceled by user or system', 5, 0, now(), now()),

      -- Gender
      (12, 'en-US', 'Male', '', 1, 0, now(), now()),
      (13, 'en-US', 'Female', '', 2, 0, now(), now()),
      (14, 'en-US', 'Unknown', 'Default value when user does not specify', 3, 0, now(), now()),

      -- Payment Method
      (15, 'en-US', 'Alipay', 'Supports Huabei and Yu’ebao', 1, 0, now(), now()),
      (16, 'en-US', 'WeChat Pay', 'Requires WeChat account binding', 2, 0, now(), now()),
      (17, 'en-US', 'UnionPay', 'Supports credit and debit cards', 3, 0, now(), now()),
      (18, 'en-US', 'Cash', 'Offline payment; deprecated as of Jan 2025', 4, 0, now(), now()),

      -- Approval Business Type
      (19, 'en-US', 'Purchase Order', 'Approval request submitted by a purchase order', 1, 0, now(), now()),
      (20, 'en-US', 'Replenishment', 'Approval request triggered by inventory replenishment', 2, 0, now(), now()),
      (21, 'en-US', 'Payment', 'Approval request triggered by finance payment', 3, 0, now(), now())
;
SELECT setval('sys_dict_entry_i18n_id_seq', (SELECT MAX(id) FROM sys_dict_entry_i18n));

-- ----------------------------
-- 插入 internal_message_categories 站内信分类
-- ----------------------------
INSERT INTO public.internal_message_categories (id, code, name, remark, sort_order, is_enabled, created_at)
VALUES
    -- 订单相关分类（原主分类+子分类平级展示）
    (1, 'order', '订单通知', '包含订单支付、发货、退款等全流程通知', 1, true, NOW()),
    (101, 'order_paid', '支付成功', '订单支付完成时触发的通知', 2, true, NOW()),
    (102, 'order_unpaid', '支付超时', '订单未在规定时间内支付的提醒', 3, true, NOW()),
    (103, 'order_shipped', '已发货', '商家发货后通知用户', 4, true, NOW()),
    (104, 'order_refunded', '已退款', '订单退款流程完成的通知', 5, true, NOW()),

    -- 系统相关分类
    (2, 'system', '系统通知', '系统公告、维护提醒、版本更新等平台级通知', 6, true, NOW()),
    (201, 'system_announcement', '系统公告', '平台规则更新、重要通知等', 7, true, NOW()),
    (202, 'system_maintenance', '维护通知', '系统计划内维护的时间提醒', 8, true, NOW()),
    (203, 'system_upgrade', '版本更新', '客户端或功能升级的提示', 9, true, NOW()),

    -- 活动相关分类
    (3, 'activity', '活动通知', '营销活动报名、开始、结束等提醒', 10, true, NOW()),
    (301, 'activity_signup', '报名成功', '用户报名活动后确认通知', 11, true, NOW()),
    (302, 'activity_start', '活动开始', '活动即将开始的倒计时提醒', 12, true, NOW()),
    (303, 'activity_end', '活动结束', '活动结束及结果公示通知', 13, true, NOW()),

    -- 用户相关分类
    (4, 'user', '用户通知', '账号安全、信息变更、权限调整等个人相关通知', 14, true, NOW()),
    (401, 'user_login_abnormal', '异地登录', '账号在陌生设备登录的安全提醒', 15, true, NOW()),
    (402, 'user_profile_updated', '资料变更', '用户手机号、邮箱等信息修改后通知', 16, true, NOW()),
    (403, 'user_permission_changed', '权限变更', '账号角色或功能权限调整通知', 17, true, NOW())
;
SELECT setval('internal_message_categories_id_seq', (SELECT MAX(id) FROM internal_message_categories));


-- ============================================================
-- ERP 演示数据（仓库/库位/库存/拣货/移动 + 采购/应付/付款/审批）
-- ============================================================
-- ----------------------------
-- inv_warehouses — 仓库（含收货库位软引用）
-- ----------------------------
INSERT INTO inv_warehouses (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, code, name, location, enable, receiving_location_id) VALUES
    (1, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'WH-BJ', '北京中心仓', '北京市朝阳区', true, 1),
    (2, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'WH-SH', '上海区域仓', '上海市浦东新区', true, 2)
;
SELECT setval(pg_get_serial_sequence('public.inv_warehouses','id'), COALESCE((SELECT MAX(id) FROM inv_warehouses),1), true);
-- ----------------------------
-- inv_stock_locations — 库位（INTERNAL 仓库内 / SUPPLIER|CUSTOMER 租户级虚拟位置）
-- 注意：服务层 GetLocationID/GetSupplierLocationID/GetCustomerLocationID 均以
-- Only() 取唯一行（Odoo 式双轨库存边界）：每仓库仅一条 INTERNAL 接收位置
-- （仓库创建时自动生成），SUPPLIER/CUSTOMER 每租户各仅一条，不能多建。
-- ----------------------------
INSERT INTO inv_stock_locations (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, name, parent_id, path, usage, warehouse_code) VALUES
    (1, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, '北京中心仓-收货暂存区', NULL, '/1/', 'INTERNAL', 'WH-BJ'),
    (2, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, '上海区域仓-收货暂存区', NULL, '/2/', 'INTERNAL', 'WH-SH'),
    (3, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'supplier location', NULL, '/3/', 'SUPPLIER', NULL),
    (4, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'customer location', NULL, '/4/', 'CUSTOMER', NULL),
    (5, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'inventory loss location', NULL, '/5/', 'INVENTORY_LOSS', NULL)
;
SELECT setval(pg_get_serial_sequence('public.inv_stock_locations','id'), COALESCE((SELECT MAX(id) FROM inv_stock_locations),1), true);
-- ----------------------------
-- inv_stock_quants — 库存量（按 库位×产品 唯一）
-- ----------------------------
INSERT INTO inv_stock_quants (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, location_id, product_code, quantity) VALUES
    (1, now() - interval '9 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 1, 'SKU-A001', 1000),
    (2, now() - interval '30 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 2, 'SKU-A003', 50),
    (3, now() - interval '30 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 1, 'SKU-A002', 200)
;
SELECT setval(pg_get_serial_sequence('public.inv_stock_quants','id'), COALESCE((SELECT MAX(id) FROM inv_stock_quants),1), true);
-- ----------------------------
-- inv_stock_pickings — 拣货单（INCOMING 入库 / INTERNAL 调拨）
-- ----------------------------
INSERT INTO inv_stock_pickings (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, picking_number, picking_type, source_location_id, destination_location_id, purchase_order_id, partner_code) VALUES
    (1, now() - interval '9 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PK-IN-2026-0001', 'INCOMING', 3, 1, 1, 'SUP-001'),
    (2, now() - interval '6 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PK-TR-2026-0002', 'INTERNAL', 1, 2, 0, NULL)
;
SELECT setval(pg_get_serial_sequence('public.inv_stock_pickings','id'), COALESCE((SELECT MAX(id) FROM inv_stock_pickings),1), true);
-- ----------------------------
-- inv_stock_moves — 库存移动计划（DONE/DRAFT 等多状态）
-- ----------------------------
INSERT INTO inv_stock_moves (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, picking_id, product_code, source_location_id, destination_location_id, planned_quantity, state, purchase_order_item_id) VALUES
    (1, now() - interval '9 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 1, 'SKU-A001', 3, 1, 1000, 'DONE', 1),
    (2, now() - interval '6 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 2, 'SKU-A001', 1, 2, 1000, 'DRAFT', 0)
;
SELECT setval(pg_get_serial_sequence('public.inv_stock_moves','id'), COALESCE((SELECT MAX(id) FROM inv_stock_moves),1), true);
-- ----------------------------
-- inv_stock_move_lines — 库存移动执行记录（仅 DONE 移动有）
-- ----------------------------
INSERT INTO inv_stock_move_lines (id, created_at, created_by, updated_by, deleted_by, remark, tenant_id, move_id, picking_id, product_code, source_location_id, destination_location_id, executed_quantity) VALUES
    (1, now() - interval '9 days', NULL, NULL, NULL, NULL, 1, 1, 1, 'SKU-A001', 3, 1, 1000)
;
SELECT setval(pg_get_serial_sequence('public.inv_stock_move_lines','id'), COALESCE((SELECT MAX(id) FROM inv_stock_move_lines),1), true);
-- ----------------------------
-- prd_products — 产品（SKU）
-- ----------------------------
INSERT INTO prd_products (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, code, name, spec, unit, enable) VALUES
    (1, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'SKU-A001', 'M8不锈钢内六角螺栓', 'M8×30mm', '个', true),
    (2, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'SKU-A002', '贴片电阻 10kΩ', '0805 1%精度', '个', true),
    (3, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'SKU-A003', 'PVC绝缘穿线管', 'DN20', '米', true)
;
SELECT setval(pg_get_serial_sequence('public.prd_products','id'), COALESCE((SELECT MAX(id) FROM prd_products),1), true);
-- ----------------------------
-- pur_suppliers — 供应商（含禁用）
-- ----------------------------
INSERT INTO pur_suppliers (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, code, name, contact, phone, enable) VALUES
    (1, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'SUP-001', '华东五金供货商', '张铭', '13800000001', true),
    (2, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'SUP-002', '南方电子元件厂', '李锐', '13800000002', true),
    (3, now(), NULL, NULL, NULL, NULL, NULL, NULL, 1, 'SUP-003', '北方建材批发站', '王磊', '13800000003', false)
;
SELECT setval(pg_get_serial_sequence('public.pur_suppliers','id'), COALESCE((SELECT MAX(id) FROM pur_suppliers),1), true);
-- ----------------------------
-- pur_purchase_orders — 采购单（多状态）
-- ----------------------------
INSERT INTO pur_purchase_orders (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, po_number, supplier_code, status, total_amount, warehouse_code) VALUES
    (1, now() - interval '10 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PO-2026-0001', 'SUP-001', 'APPROVED', 5000, 'WH-BJ'),
    (2, now() - interval '8 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PO-2026-0002', 'SUP-002', 'DRAFT', 1000, 'WH-SH'),
    (3, now() - interval '15 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PO-2026-0003', 'SUP-001', 'CANCELLED', 300, 'WH-BJ'),
    (4, now() - interval '2 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PO-2026-0004', 'SUP-002', 'SUBMITTED', 1600, 'WH-SH')
;
SELECT setval(pg_get_serial_sequence('public.pur_purchase_orders','id'), COALESCE((SELECT MAX(id) FROM pur_purchase_orders),1), true);
-- ----------------------------
-- pur_purchase_order_items — 采购单明细
-- ----------------------------
INSERT INTO pur_purchase_order_items (id, created_at, remark, tenant_id, po_id, sku_code, quantity, unit_price, amount, received_quantity) VALUES
    (1, now() - interval '10 days', NULL, 1, 1, 'SKU-A001', 1000, 5, 5000, 1000),
    (2, now() - interval '8 days', NULL, 1, 2, 'SKU-A002', 500, 2, 1000, 0),
    (3, now() - interval '15 days', NULL, 1, 3, 'SKU-A003', 100, 3, 300, 0),
    (4, now() - interval '2 days', NULL, 1, 4, 'SKU-A002', 800, 2, 1600, 0)
;
SELECT setval(pg_get_serial_sequence('public.pur_purchase_order_items','id'), COALESCE((SELECT MAX(id) FROM pur_purchase_order_items),1), true);
-- ----------------------------
-- fin_payables — 应付单（PENDING/SETTLED/CANCELLED）
-- ----------------------------
INSERT INTO fin_payables (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, payable_number, po_ref, supplier_code, amount, paid_amount, due_date, status) VALUES
    (1, now() - interval '9 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'AP-2026-0001', 'PO-2026-0001', 'SUP-001', 5000, 0, now() + interval '30 days', 'PENDING'),
    (2, now() - interval '40 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'AP-2026-0002', 'PO-2025-8812', 'SUP-002', 1000, 1000, now() - interval '5 days', 'SETTLED'),
    (3, now() - interval '15 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'AP-2026-0003', 'PO-2026-0003', 'SUP-001', 300, 0, now() + interval '15 days', 'CANCELLED')
;
SELECT setval(pg_get_serial_sequence('public.fin_payables','id'), COALESCE((SELECT MAX(id) FROM fin_payables),1), true);
-- ----------------------------
-- fin_payments — 付款（APPLIED/PENDING/REJECTED）
-- ----------------------------
INSERT INTO fin_payments (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, payment_number, payable_id, amount, method, status) VALUES
    (1, now() - interval '38 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PAY-2026-0001', 2, 1000, 'BANK_TRANSFER', 'APPLIED'),
    (2, now() - interval '1 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PAY-2026-0002', 1, 2000, 'BANK_TRANSFER', 'PENDING'),
    (3, now() - interval '3 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, 'PAY-2026-0003', 1, 500, 'CHECK', 'REJECTED')
;
SELECT setval(pg_get_serial_sequence('public.fin_payments','id'), COALESCE((SELECT MAX(id) FROM fin_payments),1), true);
-- ----------------------------
-- apr_approval_requests — 审批请求（APPROVED/PENDING/REJECTED）
-- ----------------------------
INSERT INTO apr_approval_requests (id, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by, remark, tenant_id, title, biz_type, biz_ref, summary, status, applicant_id, approver_id, comment) VALUES
    (1, now() - interval '10 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, '采购单 PO-2026-0001 审批', 'PURCHASE_ORDER', 'PO-2026-0001', '向 SUP-001 采购 M8 不锈钢内六角螺栓 1000 件，金额 50.00 元', 'APPROVED', 2, 2, '金额在授权范围内，同意'),
    (2, now() - interval '2 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, '采购单 PO-2026-0004 审批', 'PURCHASE_ORDER', 'PO-2026-0004', '向 SUP-002 采购贴片电阻 800 件，金额 16.00 元', 'PENDING', 2, NULL, ''),
    (3, now() - interval '20 days', NULL, NULL, NULL, NULL, NULL, NULL, 1, '采购单 PO-2026-9999 审批', 'PURCHASE_ORDER', 'PO-2026-9999', '大额紧急采购申请', 'REJECTED', 2, 2, '超出单笔采购限额，需总监复核后重新提交')
;
SELECT setval(pg_get_serial_sequence('public.apr_approval_requests','id'), COALESCE((SELECT MAX(id) FROM apr_approval_requests),1), true);

COMMIT;
