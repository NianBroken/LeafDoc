// 本文件保存账号的数据库读写入口。密码计算、角色授权和会话撤销由上层服务负责，
// 此处按统一列顺序读出账号，并把重复键和账号不存在转换为认证模块的错误。
package auth

import (
	"database/sql"
	"log/slog"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

const logCloseRowsFailed = "could not close rows"

// UserStore 持有账号表连接及恢复期间的停用状态。互斥锁保护连接的打开和关闭，
// 各查询在 Connect 返回后直接使用连接，不在整个查询期间持有此锁。
type UserStore struct {
	mu         sync.Mutex
	storageDir string
	db         *sql.DB
	// 恢复流程停用存储后，Connect 拒绝重新打开连接，避免后续请求继续读取账号表。
	suspended bool
}

// NewUserStore 根据存储目录对应的数据库配置打开连接。打开失败时不返回可用存储。
func NewUserStore(storageDir string) (*UserStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &UserStore{storageDir: storageDir, db: db}, nil
}

// Connect 在连接为空且存储未停用时打开数据库。已有连接直接复用，
// 因此正常 Close 后仍可由后续调用重新连接，而 suspend 后不能走此路径。
func (f *UserStore) Connect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.suspended {
		return errUserStoreUnavailable()
	}
	if f.db != nil {
		return nil
	}
	db, err := database.Open(f.storageDir)
	if err != nil {
		return err
	}
	f.db = db
	return nil
}

// suspend 标记账号存储不可用并关闭现有连接，供恢复业务记录时阻止后续连接。
// 关闭失败也清空本对象的连接指针，并把关闭错误交给调用方处理。
func (f *UserStore) suspend() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.suspended = true
	if f.db == nil {
		return nil
	}
	err := f.db.Close()
	f.db = nil
	return err
}

// Close 关闭当前连接，成功后清空指针；它不设置停用标记。
// 关闭失败时保留指针并返回错误，供调用方识别资源关闭失败。
func (f *UserStore) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.db != nil {
		err := f.db.Close()
		if err != nil {
			return err
		}
		f.db = nil
	}
	return nil
}

// CreateUser 写入账号标识、资料、角色和强制改密标记。密码直接取自传入对象，
// 此处不计算密码摘要，也不写入 must_set_password，该列使用数据库默认值。
func (f *UserStore) CreateUser(user *User) error {
	// 连接不可用时直接返回，避免继续执行插入语句。
	err := f.Connect()
	if err != nil {
		return err
	}
	// 占位符只绑定字段值，重复账号约束错误通过统一映射返回。
	_, err = f.db.Exec(`
		INSERT INTO users (id, username, password, email, role, must_change_password)
		VALUES (?, ?, ?, ?, ?, ?);
	`, user.ID, user.Username, user.Password, user.Email, user.Role, user.MustChangePassword)
	if err != nil {
		return f.mapConstraintViolationToError(err)
	}
	return nil
}

// 查询列的顺序必须与 scanUser 的目标字段顺序相同，两个密码状态分别读取。
const userColumns = `id, username, password, email, role, must_set_password, must_change_password`

// scanner 统一单行查询和多行遍历的读取接口，两种结果均通过 Scan 填入字段。
type scanner interface {
	Scan(dest ...any) error
}

// scanUser 按 userColumns 的顺序构造账号。即使扫描失败也返回分配的对象及错误，
// 各公开查询负责在错误分支丢弃该对象，避免使用不完整的字段。
func scanUser(row scanner) (*User, error) {
	user := &User{}
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role, &user.MustSetPassword, &user.MustChangePassword)
	return user, err
}

// GetUserByID 按账号标识读取完整记录；没有对应行时返回 ErrUserNotFound。
func (f *UserStore) GetUserByID(id string) (*User, error) {
	// 连接失败和查询失败分别向上传递，只有无记录使用账号不存在错误。
	err := f.Connect()
	if err != nil {
		return nil, err
	}

	// 账号标识通过参数绑定，返回字段包含密码摘要和两个密码状态标记。
	row := f.db.QueryRow(`SELECT `+userColumns+`
		FROM users
		WHERE id = ?;
	`, id)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetUserByUsername 使用数据库的用户名比较规则查找账号，不在此处裁剪或转换名称。
func (f *UserStore) GetUserByUsername(username string) (*User, error) {
	// 使用同一连接入口，让恢复期间的停用状态同样作用于名称查询。
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// 用户名原值作为查询参数，无记录与底层数据库错误分开返回。
	row := f.db.QueryRow(`SELECT `+userColumns+`
		FROM users
		WHERE username = ?;
	`, username)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetUserByEmail 按传入邮箱读取账号，不在存储层执行邮箱格式检查或大小写转换。
func (f *UserStore) GetUserByEmail(email string) (*User, error) {
	// 查询前确认连接状态，失败时不会继续访问账号表。
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// 数据库决定邮箱匹配结果；不存在时返回与其他账号查询相同的错误。
	row := f.db.QueryRow(`SELECT `+userColumns+`
		FROM users
		WHERE email = ?;
	`, email)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// UpdateUser 写入名称、密码、邮箱和角色，不更改两个密码状态标记。
// SQL 同时检查管理员数量，阻止把仅剩的管理员降为其他角色。
func (f *UserStore) UpdateUser(user *User) error {
	// 存储停用或连接失败时，资料更新不能继续。
	err := f.Connect()
	if err != nil {
		return err
	}

	// 保存更新前角色，用于判断零行更新是否属于管理员降权被拒绝。
	existingUser, err := f.GetUserByID(user.ID)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// 管理员降权限制写在同一条 UPDATE 的条件中。派生表统计管理员账号，
	// 只有原角色为管理员、目标不是管理员且数量不超过一时排除该行。
	result, err := f.db.Exec(`
		UPDATE users
		SET username = ?, password = ?, email = ?, role = ?
		WHERE id = ?
		  AND NOT (
			role = ?
			AND ? != ?
			AND (SELECT COUNT(*) FROM (SELECT id FROM users WHERE role = ?) AS administrators) <= 1
		  );
	`, user.Username, user.Password, user.Email, user.Role, user.ID, RoleAdmin, user.Role, RoleAdmin, RoleAdmin)
	if err != nil {
		return f.mapConstraintViolationToError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	// 其他零行更新不产生专用错误，例如资料内容完全相同的更新。
	if rowsAffected == 0 && existingUser.Role == RoleAdmin && user.Role != RoleAdmin {
		return ErrLastAdminCannotBeDemoted
	}
	return nil
}

// DeleteUser 确认账号存在后按标识删除。此入口自身不统计管理员数量，
// 删除权限和保留管理员的业务规则由调用它的上层服务执行。
func (f *UserStore) DeleteUser(id string) error {
	// 不可用的存储直接拒绝删除操作。
	err := f.Connect()
	if err != nil {
		return err
	}

	// 独立读取让请求开始时不存在的账号返回 ErrUserNotFound。
	_, err = f.GetUserByID(id)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// 删除语句不检查受影响行数，读取和删除之间没有在此建立事务。
	_, err = f.db.Exec(`
		DELETE FROM users
		WHERE id = ?;
	`, id)
	if err != nil {
		return err
	}
	return nil
}

// GetAdminUser 读取任意一名管理员。查询没有排序条件，不承诺固定选择某个账号。
func (f *UserStore) GetAdminUser() (*User, error) {
	// 连接错误向上传递，没有管理员记录则返回账号不存在错误。
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// LIMIT 只限制读取一行，角色条件固定为管理员。
	row := f.db.QueryRow(`SELECT ` + userColumns + `
		FROM users
		WHERE role = 'admin'
		LIMIT 1;
	`)

	user, err := scanUser(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetAllUsers 按数据库返回顺序读取全部账号，不做排序或密码字段脱敏。
// 没有记录时返回空切片的零值；遍历失败时同时返回已读取记录和遍历错误。
func (f *UserStore) GetAllUsers() ([]*User, error) {
	// 获取行游标前确认连接；查询失败时尚未取得需要关闭的游标。
	err := f.Connect()
	if err != nil {
		return nil, err
	}
	// 取得游标后始终安排关闭，关闭错误写入日志，不覆盖查询的返回值。
	rows, err := f.db.Query(`SELECT ` + userColumns + `
		FROM users;
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var users []*User
	// 单行扫描失败直接返回错误并丢弃累计结果；正常结束后检查遍历错误。
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// CountAdminUsers 只统计角色为 admin 的记录，供管理员保留规则读取当前数量。
func (f *UserStore) CountAdminUsers() (int, error) {
	err := f.Connect()
	if err != nil {
		return 0, err
	}
	row := f.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin';`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountEditorUsers 合计 admin 与 editor 两类账号，viewer 不计入结果。
// 这里返回数据库计数，不在存储层执行配额判断或阻止创建账号。
func (f *UserStore) CountEditorUsers() (int, error) {
	err := f.Connect()
	if err != nil {
		return 0, err
	}
	row := f.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role IN ('admin', 'editor');`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetUserCount 统计全部账号，不按角色或密码状态筛选。
func (f *UserStore) GetUserCount() (int, error) {
	// COUNT 查询失败时返回零和错误，零值本身不代表查询成功。
	err := f.Connect()
	if err != nil {
		return 0, err
	}
	// 聚合结果只有一个计数字段，扫描错误直接交给调用方。
	row := f.db.QueryRow(`
		SELECT COUNT(*)
		FROM users;
	`)
	var count int
	err = row.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// mapConstraintViolationToError 将数据库重复键归为账号已存在，其他错误原样返回。
func (f *UserStore) mapConstraintViolationToError(err error) error {
	if database.IsDuplicate(err) {
		return ErrUserAlreadyExists
	}
	return err
}

// UpdatePassword 保存调用方提供的密码值并清除强制改密标记。
// 此处不计算摘要、不清除 must_set_password，也不撤销账号会话。
func (f *UserStore) UpdatePassword(userID string, newPassword string) error {
	// 确认连接可用后再检查目标账号。
	err := f.Connect()
	if err != nil {
		return err
	}

	// 在执行写入前区分账号不存在与数据库读取失败。
	_, err = f.GetUserByID(userID)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUserNotFound
		}
		return err
	}

	// 密码字段和 must_change_password 在同一条语句中更新。
	_, err = f.db.Exec(`
		UPDATE users
		SET password = ?, must_change_password = 0
		WHERE id = ?;
	`, newPassword, userID)
	if err != nil {
		return err
	}
	return nil
}

// SetMustSetPassword 只修改首次设密标记，先确认账号存在，再把布尔值映射为整数。
// 密码内容和强制改密标记均不受这条语句影响。
func (f *UserStore) SetMustSetPassword(userID string, value bool) error {
	if err := f.Connect(); err != nil {
		return err
	}
	if _, err := f.GetUserByID(userID); err != nil {
		return err
	}

	v := 0
	if value {
		v = 1
	}
	_, err := f.db.Exec(`
		UPDATE users
		SET must_set_password = ?
		WHERE id = ?;
	`, v, userID)
	return err
}
