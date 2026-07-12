// xiuno-go v2.1.0-beta 尼克修改版
package core

// Policy 统一权限判定中心
// 将散落在原版 PHP 各处的 if ($gid == 1 || $uid == xxx) 收敛至此
// 未来扩展版主（Moderator）逻辑时，只需修改此文件
//
// 设计约束：Policy 位于 core 包，不能导入 model 包（model 已导入 core，防止循环依赖）
// 因此所有参数使用基本类型（uint32 uid, uint16 gid），而非 *model.User
type Policy struct{}

// CanManageThread 判断用户是否有权编辑/删除帖子
// 规则：
//   1. uid == 0 的游客绝对无权
//   2. 超级管理员（gid == 1）拥有一切权限
//   3. 作者本人（uid == threadUID）有权操作自己的帖子
//   4. （未来扩展）版主有权管理所属版块的帖子
func (p *Policy) CanManageThread(uid uint32, gid uint16, threadUID uint32, fid uint32) bool {
	// 1. 游客绝对无权
	if uid == 0 {
		return false
	}
	// 2. 超级管理员（GID = 1）拥有一切权限
	if gid == 1 {
		return true
	}
	// 3. 作者本人有权操作自己的帖子
	if uid == threadUID {
		return true
	}
	// 4. （未来扩展）判断 user 是否为该 fid 的版主
	// ...

	return false
}

// CanManagePost 判断用户是否有权编辑/删除回帖
func (p *Policy) CanManagePost(uid uint32, gid uint16, postUID uint32, tid uint32) bool {
	if uid == 0 {
		return false
	}
	if gid == 1 {
		return true
	}
	if uid == postUID {
		return true
	}
	return false
}

// CanModerateThread 判断用户是否有权执行版务操作（置顶/关闭）
// 原版中 GID 1(超管), 2(超版), 4(版主), 5(实习版主) 具有版务权限
func (p *Policy) CanModerateThread(uid uint32, gid uint16) bool {
	if uid == 0 {
		return false
	}
	return gid == 1 || gid == 2 || gid == 4 || gid == 5
}

// CanDeleteUser 判断用户是否有权删除其他用户
// 原版：$group['allowdeleteuser']，对应 GID 1(超管), 2(超版)
// 保护规则：不能删除 gid < 6 的管理组成员
func (p *Policy) CanDeleteUser(uid uint32, gid uint16, targetGID uint16) bool {
	if uid == 0 {
		return false
	}
	// 只有超管和超版可以删除用户
	if gid != 1 && gid != 2 {
		return false
	}
	// 不能删除管理组成员（gid < 6）
	if targetGID < 6 {
		return false
	}
	return true
}
