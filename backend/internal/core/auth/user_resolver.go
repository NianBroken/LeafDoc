// 本文件为页面作者等引用提供用户显示标签缓存，并支持整体重新加载。
package auth

import (
	"errors"
	"sync"
)

// UserLabel 仅保存用户 ID 和用户名，不携带邮箱、角色或密码信息。
type UserLabel struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// UserResolver 缓存标签及已确认不存在的用户，使用读写锁保护缓存映射。
type UserResolver struct {
	// 查询未命中和整体加载时调用提供器获取当前服务，已缓存结果不会自动重新查询。
	userService func() *UserService
	resolved    map[string]*UserLabel
	mu          sync.RWMutex
}

// NewUserResolver 读取全部用户预填标签，查询失败直接返回，不创建不完整解析器。
func NewUserResolver(userService func() *UserService) (*UserResolver, error) {
	users, err := userService().GetUsers() // 构造阶段预载当前用户列表。
	if err != nil {
		// 错误向装配调用方传递，此处不重复记录同一失败。
		return nil, err
	}
	r := &UserResolver{
		userService: userService,
		resolved:    make(map[string]*UserLabel),
	}

	for _, user := range users {
		r.resolved[user.ID] = &UserLabel{
			ID:       user.ID,
			Username: user.Username,
		}
	}

	return r, nil
}

// ResolveUserLabel 空 ID 返回 nil，缓存命中直接返回其中指针，未命中再访问用户存储。
// 缓存缺失用户的首次查询返回 ErrUserNotFound，后续命中 nil 缓存时返回 nil 且无错误。
func (r *UserResolver) ResolveUserLabel(userID string) (*UserLabel, error) {
	if userID == "" {
		return nil, nil
	}

	// 读锁只保护映射查找，返回标签对象不额外复制。
	r.mu.RLock()
	if user, ok := r.resolved[userID]; ok {
		r.mu.RUnlock()
		return user, nil
	}
	r.mu.RUnlock()

	// 数据库查询在缓存锁外执行，不阻塞其他命中缓存的读取。
	user, err := r.userService().GetUserByID(userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// 只缓存已确认不存在的用户，减少已删除作者等引用的重复数据库查询。
			// 加锁后再次检查，避免覆盖并发查询已写入的结果。
			r.mu.Lock()
			if _, ok := r.resolved[userID]; !ok {
				r.resolved[userID] = nil
			}
			r.mu.Unlock()
		}
		// 其他存储错误不写入缓存，下次未命中仍会重新查询。
		return nil, err
	}
	label := &UserLabel{ID: user.ID, Username: user.Username}

	// 写入前再次检查缓存，并发流程已写入的结果优先于当前查询结果。
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.resolved[userID]; ok {
		return existing, nil
	}

	r.resolved[userID] = label
	return label, nil
}

// Reload 先在锁外完整读取并建立新映射，成功后一次替换全部缓存。
// 读取失败保留旧映射，成功加载也会清除此前缓存的不存在记录。
func (r *UserResolver) Reload() error {
	users, err := r.userService().GetUsers()
	if err != nil {
		return err
	}

	newMap := make(map[string]*UserLabel, len(users))
	for _, u := range users {
		newMap[u.ID] = &UserLabel{ID: u.ID, Username: u.Username}
	}

	r.mu.Lock()
	r.resolved = newMap
	r.mu.Unlock()

	return nil
}
