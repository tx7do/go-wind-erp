package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"path"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/go-utils/trans"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/sliceutil"

	"go-wind-erp/app/core/service/internal/data"

	authenticationV1 "go-wind-erp/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-erp/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-erp/api/gen/go/permission/service/v1"
	storageV1 "go-wind-erp/api/gen/go/storage/service/v1"

	"go-wind-erp/pkg/constants"
	"go-wind-erp/pkg/netutil"
	"go-wind-erp/pkg/oss"
	appViewer "go-wind-erp/pkg/entgo/viewer"
)

// viewerUserIDFromContext 从 viewer context 提取操作人的用户 ID。
// core 服务经由 ent middleware 注入 viewer context（来源为 gRPC metadata 中的 OperatorMetadata）。
// 系统上下文（初始化、定时任务等）不存在 viewer，返回 (0, false)。
func viewerUserIDFromContext(ctx context.Context) (uint32, bool) {
	vc, exist := viewer.FromContext(ctx)
	if !exist || vc == nil {
		return 0, false
	}
	uid := vc.UserID()
	if uid == 0 {
		return 0, false
	}
	return uint32(uid), true
}

type UserService struct {
	identityV1.UnimplementedUserServiceServer

	log *log.Helper

	userRepo           data.UserRepo
	userCredentialRepo *data.UserCredentialRepo

	roleRepo     *data.RoleRepo
	positionRepo *data.PositionRepo
	orgUnitRepo  *data.OrgUnitRepo
	tenantRepo   *data.TenantRepo

	membershipRepo *data.MembershipRepo

	// 头像上传（MinIO + file 元数据）与联系方式验证码所需依赖。
	mc                *oss.MinIOClient
	fileRepo          *data.FileRepo
	contactCodeCache  *data.ContactCodeCache
	contactCodeSender ContactCodeSender
}

func NewUserService(
	ctx *bootstrap.Context,
	userRepo data.UserRepo,
	roleRepo *data.RoleRepo,
	userCredentialRepo *data.UserCredentialRepo,
	positionRepo *data.PositionRepo,
	orgUnitRepo *data.OrgUnitRepo,
	tenantRepo *data.TenantRepo,
	membershipRepo *data.MembershipRepo,
	mc *oss.MinIOClient,
	fileRepo *data.FileRepo,
	contactCodeCache *data.ContactCodeCache,
	contactCodeSender ContactCodeSender,
) *UserService {
	svc := &UserService{
		log:                ctx.NewLoggerHelper("user/service/core-service"),
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		userCredentialRepo: userCredentialRepo,
		positionRepo:       positionRepo,
		orgUnitRepo:        orgUnitRepo,
		tenantRepo:         tenantRepo,
		membershipRepo:     membershipRepo,
		mc:                  mc,
		fileRepo:            fileRepo,
		contactCodeCache:    contactCodeCache,
		contactCodeSender:   contactCodeSender,
	}

	svc.init()

	return svc
}

func (s *UserService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if count, _ := s.userRepo.Count(ctx, nil); count == 0 {
		_ = s.createDefaultUser(ctx)
	}
}

func (s *UserService) extractRelationIDs(
	users []*identityV1.User,
	roleSet aggregator.ResourceMap[uint32, *permissionV1.Role],
	tenantSet aggregator.ResourceMap[uint32, *identityV1.Tenant],
	orgUnitSet aggregator.ResourceMap[uint32, *identityV1.OrgUnit],
	posSet aggregator.ResourceMap[uint32, *identityV1.Position],
) {
	for _, v := range users {
		if v == nil {
			continue
		}

		if id := v.GetTenantId(); id > 0 {
			tenantSet[id] = nil
		}

		for _, roleId := range v.RoleIds {
			if roleId > 0 {
				roleSet[roleId] = nil
			}
		}

		if v.GetOrgUnitId() > 0 {
			orgUnitSet[v.GetOrgUnitId()] = nil
		}
		if len(v.OrgUnitIds) > 0 {
			for _, orgID := range v.OrgUnitIds {
				if orgID > 0 {
					orgUnitSet[orgID] = nil
				}
			}
		}

		if v.GetPositionId() > 0 {
			posSet[v.GetPositionId()] = nil
		}
		if len(v.PositionIds) > 0 {
			for _, posID := range v.PositionIds {
				if posID > 0 {
					posSet[posID] = nil
				}
			}
		}

	}
}

func (s *UserService) fetchRelationInfo(
	ctx context.Context,
	roleSet aggregator.ResourceMap[uint32, *permissionV1.Role],
	tenantSet aggregator.ResourceMap[uint32, *identityV1.Tenant],
	orgUnitSet aggregator.ResourceMap[uint32, *identityV1.OrgUnit],
	posSet aggregator.ResourceMap[uint32, *identityV1.Position],
) error {
	if len(roleSet) > 0 {
		roleIds := make([]uint32, 0, len(roleSet))
		for id := range roleSet {
			roleIds = append(roleIds, id)
		}

		roles, err := s.roleRepo.ListRolesByRoleIds(ctx, roleIds)
		if err != nil {
			s.log.Errorf("query roles err: %v", err)
			return err
		}

		for _, role := range roles {
			roleSet[role.GetId()] = role
		}
	}

	if len(tenantSet) > 0 {
		tenantIds := make([]uint32, 0, len(tenantSet))
		for id := range tenantSet {
			tenantIds = append(tenantIds, id)
		}

		tenants, err := s.tenantRepo.ListTenantsByIds(ctx, tenantIds)
		if err != nil {
			s.log.Errorf("query tenants err: %v", err)
			return err
		}

		for _, tenant := range tenants {
			tenantSet[tenant.GetId()] = tenant
		}
	}

	if len(orgUnitSet) > 0 {
		orgUnitIds := make([]uint32, 0, len(orgUnitSet))
		for id := range orgUnitSet {
			orgUnitIds = append(orgUnitIds, id)
		}

		orgUnits, err := s.orgUnitRepo.ListOrgUnitsByIds(ctx, orgUnitIds)
		if err != nil {
			s.log.Errorf("query orgUnits err: %v", err)
			return err
		}

		for _, orgUnit := range orgUnits {
			orgUnitSet[orgUnit.GetId()] = orgUnit
		}
	}

	if len(posSet) > 0 {
		posIds := make([]uint32, 0, len(posSet))
		for id := range posSet {
			posIds = append(posIds, id)
		}

		positions, err := s.positionRepo.ListPositionByIds(ctx, posIds)
		if err != nil {
			s.log.Errorf("query positions err: %v", err)
			return err
		}

		for _, position := range positions {
			posSet[position.GetId()] = position
		}
	}

	return nil
}

func (s *UserService) bindRelations(
	users []*identityV1.User,
	roleSet aggregator.ResourceMap[uint32, *permissionV1.Role],
	tenantSet aggregator.ResourceMap[uint32, *identityV1.Tenant],
	orgUnitSet aggregator.ResourceMap[uint32, *identityV1.OrgUnit],
	posSet aggregator.ResourceMap[uint32, *identityV1.Position],
) {
	aggregator.PopulateMulti(
		users,
		roleSet,
		func(ou *identityV1.User) []uint32 { return ou.GetRoleIds() },
		func(ou *identityV1.User, r []*permissionV1.Role) {
			for _, role := range r {
				ou.RoleNames = append(ou.RoleNames, role.GetName())
				ou.Roles = append(ou.Roles, role.GetCode())
			}
		},
	)
	aggregator.Populate(
		users,
		roleSet,
		func(ou *identityV1.User) uint32 { return ou.GetRoleId() },
		func(ou *identityV1.User, r *permissionV1.Role) {
			ou.RoleNames = append(ou.RoleNames, r.GetName())
			ou.Roles = append(ou.Roles, r.GetCode())
		},
	)

	aggregator.Populate(
		users,
		tenantSet,
		func(ou *identityV1.User) uint32 { return ou.GetTenantId() },
		func(ou *identityV1.User, r *identityV1.Tenant) {
			ou.TenantName = r.Name
		},
	)

	aggregator.PopulateMulti(
		users,
		posSet,
		func(ou *identityV1.User) []uint32 { return ou.GetPositionIds() },
		func(ou *identityV1.User, r []*identityV1.Position) {
			for _, pos := range r {
				ou.PositionNames = append(ou.PositionNames, pos.GetName())
			}
		},
	)
	aggregator.Populate(
		users,
		posSet,
		func(ou *identityV1.User) uint32 { return ou.GetPositionId() },
		func(ou *identityV1.User, r *identityV1.Position) {
			ou.PositionName = r.Name
		},
	)

	aggregator.PopulateMulti(
		users,
		orgUnitSet,
		func(ou *identityV1.User) []uint32 { return ou.GetOrgUnitIds() },
		func(ou *identityV1.User, orgs []*identityV1.OrgUnit) {
			for _, org := range orgs {
				ou.OrgUnitNames = append(ou.OrgUnitNames, org.GetName())
			}
		},
	)
	aggregator.Populate(
		users,
		orgUnitSet,
		func(ou *identityV1.User) uint32 { return ou.GetOrgUnitId() },
		func(ou *identityV1.User, org *identityV1.OrgUnit) {
			ou.OrgUnitName = org.Name
		},
	)
}

func (s *UserService) enrichRelations(ctx context.Context, users []*identityV1.User) error {
	var roleSet = make(aggregator.ResourceMap[uint32, *permissionV1.Role])
	var tenantSet = make(aggregator.ResourceMap[uint32, *identityV1.Tenant])
	var orgUnitSet = make(aggregator.ResourceMap[uint32, *identityV1.OrgUnit])
	var posSet = make(aggregator.ResourceMap[uint32, *identityV1.Position])

	s.extractRelationIDs(users, roleSet, tenantSet, orgUnitSet, posSet)
	if err := s.fetchRelationInfo(ctx, roleSet, tenantSet, orgUnitSet, posSet); err != nil {
		return err
	}
	s.bindRelations(users, roleSet, tenantSet, orgUnitSet, posSet)
	return nil
}

func (s *UserService) queryUserIDsByRelationIDs(ctx context.Context, roleIDs []uint32, orgUnitIDs []uint32, positionIDs []uint32) ([]uint32, error) {
	if len(roleIDs) == 0 && len(orgUnitIDs) == 0 && len(positionIDs) == 0 {
		return nil, nil
	}

	switch constants.DefaultUserTenantRelationType {
	default:
		fallthrough
	case constants.UserTenantRelationOneToOne:
		return s.queryUserIDsByRelationIDsUserTenantRelationOneToOne(ctx, roleIDs, orgUnitIDs, positionIDs)
	case constants.UserTenantRelationOneToMany:
		return s.queryUserIDsByRelationIDsUserTenantRelationOneToMany(ctx, roleIDs, orgUnitIDs, positionIDs)
	}
}

func (s *UserService) queryUserIDsByRelationIDsUserTenantRelationOneToMany(_ context.Context, _, _, _ []uint32) ([]uint32, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *UserService) queryUserIDsByRelationIDsUserTenantRelationOneToOne(ctx context.Context, roleIDs, orgUnitIDs, positionIDs []uint32) ([]uint32, error) {
	if len(roleIDs) == 0 && len(orgUnitIDs) == 0 && len(positionIDs) == 0 {
		return nil, nil
	}

	var err error

	var orgUnitUserIDs []uint32
	var positionUserIDs []uint32
	var roleUserIDs []uint32
	if len(orgUnitIDs) > 0 {
		orgUnitUserIDs, err = s.userRepo.ListUserIDsByOrgUnitIDs(ctx, orgUnitIDs, true)
		if err != nil {
			return nil, err
		}
	}
	if len(positionIDs) > 0 {
		positionUserIDs, err = s.userRepo.ListUserIDsByPositionIDs(ctx, positionIDs, true)
		if err != nil {
			return nil, err
		}
	}
	if len(roleIDs) > 0 {
		roleUserIDs, err = s.userRepo.ListUserIDsByRoleIDs(ctx, roleIDs, true)
		if err != nil {
			return nil, err
		}
	}

	// 收集所有非空列表用于求交集
	lists := make([][]uint32, 0, 3)
	if orgUnitUserIDs != nil {
		lists = append(lists, orgUnitUserIDs)
	}
	if positionUserIDs != nil {
		lists = append(lists, positionUserIDs)
	}
	if roleUserIDs != nil {
		lists = append(lists, roleUserIDs)
	}

	// 如果没有任何实际列表（例如对应 ids 为空导致查询未执行），返回空
	if len(lists) == 0 {
		return []uint32{}, nil
	}

	// 逐步求交集
	result := lists[0]
	for i := 1; i < len(lists); i++ {
		result = sliceutil.Intersection(result, lists[i])
		if len(result) == 0 {
			break
		}
	}

	return result, nil
}

func (s *UserService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListUserResponse, error) {
	if req == nil {
		s.log.Errorf("invalid parameter: nil request")
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	resp, err := s.userRepo.List(ctx, req)
	if err != nil {
		s.log.Errorf("userRepo.List failed: %s", err.Error())
		return nil, err
	}

	return resp, nil
}

func (s *UserService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.CountUserResponse, error) {
	count, err := s.userRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &identityV1.CountUserResponse{
		Count: uint64(count),
	}, nil
}

func (s *UserService) Get(ctx context.Context, req *identityV1.GetUserRequest) (*identityV1.User, error) {
	resp, err := s.userRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*identityV1.User{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

func (s *UserService) Create(ctx context.Context, req *identityV1.CreateUserRequest) (result *identityV1.User, err error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	// 用户和凭证创建必须在同一事务中，避免凭证创建失败时产生孤立用户行
	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { s.userRepo.FinishTx(tx, err) }()

	// 创建用户
	var user *identityV1.User
	if user, err = s.userRepo.CreateWithTx(ctx, tx, req.GetData()); err != nil {
		return nil, err
	}

	if len(req.GetPassword()) == 0 {
		// 如果没有设置密码，则设置为默认密码。
		req.Password = trans.Ptr(constants.DefaultUserPassword)
	}

	if len(req.GetPassword()) > 0 {
		if err = s.userCredentialRepo.CreateWithTx(ctx, tx, &authenticationV1.UserCredential{
			UserId:   user.Id,
			TenantId: user.TenantId,

			IdentityType: authenticationV1.UserCredential_USERNAME.Enum(),
			Identifier:   req.Data.Username,

			CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
			Credential:     req.Password,

			IsPrimary: trans.Ptr(true),
			Status:    authenticationV1.UserCredential_ENABLED.Enum(),
		}); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *UserService) Update(ctx context.Context, req *identityV1.UpdateUserRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	// 更新用户
	if err := s.userRepo.Update(ctx, req); err != nil {
		s.log.Error(err)
		return nil, err
	}

	if len(req.GetPassword()) > 0 {
		// 通过目标用户 ID 查询其当前用户名，避免使用客户端可改的 req.Data.Username
		// （重命名场景下新名尚不存在，或撞名会误改他人密码）
		target, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
			QueryBy: &identityV1.GetUserRequest_Id{
				Id: req.GetId(),
			},
		})
		if err != nil {
			return nil, err
		}

		if err := s.userCredentialRepo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
			IdentityType:  authenticationV1.UserCredential_USERNAME,
			Identifier:    target.GetUsername(),
			NewCredential: req.GetPassword(),
			NeedDecrypt:   false,
		}); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *UserService) Delete(ctx context.Context, req *identityV1.DeleteUserRequest) (*emptypb.Empty, error) {
	// 获取操作人信息（从 viewer context 提取，core 服务经由 ent middleware 注入）
	operatorID, hasOperator := viewerUserIDFromContext(ctx)
	if !hasOperator {
		return nil, identityV1.ErrorBadRequest("operator context required to delete user")
	}

	// 获取将被删除的用户信息
	target, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: req.GetId(),
		},
	})
	if err != nil {
		return nil, err
	}

	// 禁止删除默认超级管理员：初始化时创建的平台级 admin（恒为 id=1，
	// 即便后续改名也由 id 兜底保护）。误删会导致系统失去超级管理员且无法自动重建。
	if target.GetId() == 1 ||
		(target.GetUsername() == constants.DefaultAdminUserName && target.GetTenantId() == 0) {
		s.log.Errorf("operator [%d] attempted to delete default admin user [%d]",
			operatorID, target.GetId())
		return nil, identityV1.ErrorBadRequest("default admin cannot be deleted")
	}

	// 禁止删除自己：误删自身账号将导致当前会话立即失去管理能力。
	if target.GetId() == operatorID {
		s.log.Errorf("operator [%d] attempted to delete self", operatorID)
		return nil, identityV1.ErrorBadRequest("cannot delete yourself")
	}

	err = s.userRepo.Delete(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *UserService) UserExists(ctx context.Context, req *identityV1.UserExistsRequest) (*identityV1.UserExistsResponse, error) {
	return s.userRepo.UserExists(ctx, req)
}

// EditUserPassword 修改用户密码
func (s *UserService) EditUserPassword(ctx context.Context, req *identityV1.EditUserPasswordRequest) (*emptypb.Empty, error) {
	// 获取操作者的用户信息
	u, err := s.userRepo.Get(ctx, &identityV1.GetUserRequest{
		QueryBy: &identityV1.GetUserRequest_Id{
			Id: req.GetUserId(),
		},
	})
	if err != nil {
		return nil, err
	}

	if err = s.userCredentialRepo.ResetCredential(ctx, &authenticationV1.ResetCredentialRequest{
		IdentityType:  authenticationV1.UserCredential_USERNAME,
		Identifier:    u.GetUsername(),
		NewCredential: req.GetNewPassword(),
		NeedDecrypt:   false,
	}); err != nil {
		s.log.Errorf("reset user password err: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// createDefaultUser 创建默认用户，即超级用户
func (s *UserService) createDefaultUser(ctx context.Context) error {
	var err error

	// 创建默认用户
	for _, user := range constants.DefaultUsers {
		if _, err = s.userRepo.Create(ctx, &identityV1.CreateUserRequest{
			Data: user,
		}); err != nil {
			s.log.Errorf("create default user err: %v", err)
			return err
		}
	}

	// 创建默认用户凭证
	for _, userCredential := range constants.DefaultUserCredentials {
		if err = s.userCredentialRepo.Create(ctx, &authenticationV1.CreateUserCredentialRequest{
			Data: userCredential,
		}); err != nil {
			s.log.Errorf("create default user credential err: %v", err)
			return err
		}
	}

	switch constants.DefaultUserTenantRelationType {
	default:
		fallthrough
	case constants.UserTenantRelationOneToOne:
		// 创建默认用户角色关联关系
		for _, userRole := range constants.DefaultUserRoles {
			if err = s.userRepo.AssignUserRole(ctx, userRole); err != nil {
				s.log.Errorf("create default user role relation err: %v", err)
				return err
			}
		}

	case constants.UserTenantRelationOneToMany:
		// 创建默认用户租户关联关系
		for _, membership := range constants.DefaultMemberships {
			if err = s.membershipRepo.AssignTenantMembershipWith(ctx, membership); err != nil {
				s.log.Errorf("create default user membership err: %v", err)
				return err
			}
		}
	}

	return err
}

// parseAvatarKey 将 MinIO 对象 key 拆分为目录/文件名/扩展名，与
// FileTransferService.parseKey 行为一致。
func parseAvatarKey(key string) (folder, filename, ext string) {
	folder = "/"
	if key == "" {
		return
	}
	idx := strings.LastIndex(key, "/")
	var name string
	if idx >= 0 {
		folder = key[:idx]
		name = key[idx+1:]
	} else {
		name = key
	}
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		ext = name[dot+1:]
		name = name[:dot]
	}
	_ = path.Clean(folder)
	return folder, name, ext
}

// recordAvatarAsset 将 MinIO 上传结果落库为 file 元数据记录，与
// FileTransferService.recordFile 行为一致。
func (s *UserService) recordAvatarAsset(
	ctx context.Context,
	tenantID, userID uint32,
	sourceFileName string,
	info minio.UploadInfo,
	downloadUrl string,
) error {
	dir, fileName, ext := parseAvatarKey(info.Key)
	if _, err := s.fileRepo.Create(ctx, &storageV1.CreateFileRequest{
		Data: &storageV1.File{
			Provider:      trans.Ptr(storageV1.OSSProvider_MINIO),
			BucketName:    trans.Ptr(info.Bucket),
			SaveFileName:  trans.Ptr(fileName + "." + ext),
			FileDirectory: trans.Ptr(dir),
			FileName:      trans.Ptr(sourceFileName),
			Extension:     trans.Ptr(ext),
			FileGuid:      trans.Ptr(uuid.New().String()),
			Size:          trans.Ptr(uint64(info.Size)),
			LinkUrl:       trans.Ptr(downloadUrl),
			CreatedBy:     trans.Ptr(userID),
			TenantId:      trans.Ptr(tenantID),
		},
	}); err != nil {
		s.log.Errorf("Failed to create avatar file record: %v", err)
		return err
	}
	return nil
}

// fetchAvatarBytes 从 oneof（base64 或远端 URL）取出头像字节流。
func (s *UserService) fetchAvatarBytes(ctx context.Context, req *identityV1.UploadAvatarRequest) ([]byte, error) {
	switch src := req.GetSource().(type) {
	case *identityV1.UploadAvatarRequest_ImageBase64:
		raw, err := base64.StdEncoding.DecodeString(src.ImageBase64)
		if err != nil {
			return nil, identityV1.ErrorBadRequest("invalid base64 avatar data")
		}
		return raw, nil
	case *identityV1.UploadAvatarRequest_ImageUrl:
		parsed, err := netutil.ValidateURL(src.ImageUrl)
		if err != nil {
			return nil, identityV1.ErrorBadRequest("%s", err)
		}
		httpReq, err := http.NewRequestWithContext(ctx, "GET", parsed.String(), nil)
		if err != nil {
			return nil, identityV1.ErrorBadRequest("%s", err)
		}
		client := netutil.SafeHTTPClient()
		resp, err := client.Do(httpReq)
		if err != nil {
			return nil, identityV1.ErrorBadRequest("%s", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, identityV1.ErrorBadRequest("unexpected status fetching avatar: %s", resp.Status)
		}
		data, err := io.ReadAll(netutil.LimitReader(resp.Body))
		if err != nil {
			return nil, identityV1.ErrorBadRequest("%s", err)
		}
		return data, nil
	default:
		return nil, identityV1.ErrorBadRequest("unknown avatar source")
	}
}

// UploadAvatar 将当前操作人的头像上传至 MinIO，落 file 元数据后把
// 下载 URL 写回 user.avatar 字段。
func (s *UserService) UploadAvatar(ctx context.Context, req *identityV1.UploadAvatarRequest) (*identityV1.UploadAvatarResponse, error) {
	uid, hasUser := viewerUserIDFromContext(ctx)
	tid := uint32(0)
	hasTenant := false
	if vc, exist := viewer.FromContext(ctx); exist && vc != nil {
		tid = uint32(vc.TenantID())
		hasTenant = tid != 0
	}
	if !hasTenant || !hasUser {
		return nil, identityV1.ErrorBadRequest("missing operator identity")
	}

	data, err := s.fetchAvatarBytes(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, identityV1.ErrorBadRequest("empty avatar data")
	}

	mime := http.DetectContentType(data)
	bucket := oss.ContentTypeToBucketName(mime)
	object := oss.EnsureObjectName("", "avatar", mime, oss.GenerateFileNameTypeUUID)
	reader := bytes.NewReader(data)

	info, _, downloadUrl, err := s.mc.UploadFile(ctx, bucket, object, mime, reader, int64(len(data)))
	if err != nil {
		return nil, err
	}
	if err = s.recordAvatarAsset(ctx, tid, uid, "avatar", info, downloadUrl); err != nil {
		return nil, err
	}

	// 将 URL 写回当前用户 avatar 字段。
	if err = s.userRepo.Update(ctx, &identityV1.UpdateUserRequest{
		Id:   uid,
		Data: &identityV1.User{Avatar: trans.Ptr(downloadUrl)},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"avatar"},
		},
	}); err != nil {
		return nil, err
	}

	return &identityV1.UploadAvatarResponse{Url: downloadUrl}, nil
}

// DeleteAvatar 清空当前操作人的 user.avatar 字段。
func (s *UserService) DeleteAvatar(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	uid, hasUser := viewerUserIDFromContext(ctx)
	if !hasUser {
		return nil, identityV1.ErrorBadRequest("missing operator identity")
	}
	// avatar 在 mask 中且 Data 中为 nil → 落 NULL。
	if err := s.userRepo.Update(ctx, &identityV1.UpdateUserRequest{
		Id:         uid,
		Data:       &identityV1.User{},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"avatar"}},
	}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// VerifyContact 向手机号/邮箱发送验证码并建立会话。
func (s *UserService) VerifyContact(ctx context.Context, req *identityV1.VerifyContactRequest) (*emptypb.Empty, error) {
	dest := ""
	switch c := req.GetContact().(type) {
	case *identityV1.VerifyContactRequest_Phone:
		if c.Phone == nil {
			return nil, identityV1.ErrorBadRequest("missing phone verification payload")
		}
		dest = c.Phone.GetPhone()
	case *identityV1.VerifyContactRequest_Email:
		if c.Email == nil {
			return nil, identityV1.ErrorBadRequest("missing email verification payload")
		}
		dest = c.Email.GetEmail()
	default:
		return nil, identityV1.ErrorBadRequest("unknown contact type")
	}
	if dest == "" {
		return nil, identityV1.ErrorBadRequest("empty contact destination")
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return nil, err
	}
	if err = s.contactCodeCache.Store(ctx, dest, code); err != nil {
		return nil, err
	}
	if err = s.contactCodeSender.Send(ctx, dest, code); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// BindContact 校验验证码，通过后写入 user.mobile/email 字段。
func (s *UserService) BindContact(ctx context.Context, req *identityV1.BindContactRequest) (*emptypb.Empty, error) {
	uid, hasUser := viewerUserIDFromContext(ctx)
	if !hasUser {
		return nil, identityV1.ErrorBadRequest("missing operator identity")
	}

	var dest, code, field string
	switch c := req.GetContact().(type) {
	case *identityV1.BindContactRequest_Phone:
		if c.Phone == nil {
			return nil, identityV1.ErrorBadRequest("missing phone bind payload")
		}
		dest = c.Phone.GetPhone()
		code = c.Phone.GetCode()
		field = "mobile"
	case *identityV1.BindContactRequest_Email:
		if c.Email == nil {
			return nil, identityV1.ErrorBadRequest("missing email bind payload")
		}
		dest = c.Email.GetEmail()
		code = c.Email.GetVerificationCode()
		field = "email"
	default:
		return nil, identityV1.ErrorBadRequest("unknown contact type")
	}
	if dest == "" || code == "" {
		return nil, identityV1.ErrorBadRequest("empty contact or code")
	}
	if !s.contactCodeCache.Verify(ctx, dest, code) {
		return nil, identityV1.ErrorBadRequest("verification code invalid or expired")
	}

	// 校验通过：写回对应字段。
	data := &identityV1.User{}
	if field == "mobile" {
		data.Mobile = trans.Ptr(dest)
	} else {
		data.Email = trans.Ptr(dest)
	}
	if err := s.userRepo.Update(ctx, &identityV1.UpdateUserRequest{
		Id:         uid,
		Data:       data,
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{field}},
	}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// generateNumericCode 生成定长数字验证码。
func generateNumericCode(length int) (string, error) {
	const digits = "0123456789"
	max := big.NewInt(int64(len(digits)))
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = digits[n.Int64()]
	}
	return string(b), nil
}
