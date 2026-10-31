// 本文件将品牌外观配置接到通用数据库设置存储，并提供资源目录位置。
package branding

import (
	"fmt"
	"path/filepath"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
)

// BrandingStore 缓存 MySQL 外观记录，storageDir 同时作为磁盘资源目录的父路径。
type BrandingStore struct {
	storageDir string
	store      *settings.Store[BrandingConfig]
}

// NewBrandingStore 加载 branding.json 数据库记录，缺失时初始化默认值。
// 加载与保存缓存后注入上传限制，限制本身不写入 JSON。
func NewBrandingStore(storageDir string) (*BrandingStore, error) {
	st, err := settings.New(storageDir, "branding.json", 0o644, *DefaultBrandingConfig(), injectBrandingConstraints)
	if err != nil {
		return nil, fmt.Errorf("failed to load branding config: %w", err)
	}
	return &BrandingStore{storageDir: storageDir, store: st}, nil
}

// injectBrandingConstraints 用独立默认配置中的限制替换内存字段。
// 该回调由通用 Store 在加载、重载和 Put 缓存后调用，不读取持久化限制。
func injectBrandingConstraints(cfg *BrandingConfig) {
	cfg.BrandingConstraints = DefaultBrandingConfig().BrandingConstraints
}

// brandingAssetsDir 返回实例目录下 branding 路径，不在查询时创建目录。
func (s *BrandingStore) brandingAssetsDir() string {
	return filepath.Join(s.storageDir, "branding")
}

// Load 返回配置结构副本的指针，嵌套限制映射仍与缓存共享，不执行深拷贝。
func (s *BrandingStore) Load() *BrandingConfig {
	cfg := s.store.Get()
	return &cfg
}

// Save 解引用配置后交给 Store 保存，失败添加品牌配置上下文。
// config 必须非 nil，本方法不单独检查名称和资源文件是否有效。
func (s *BrandingStore) Save(config *BrandingConfig) error {
	if err := s.store.Put(*config); err != nil {
		return fmt.Errorf("failed to write branding config: %w", err)
	}
	return nil
}

// Reload 从 MySQL 刷新缓存并经通用存储发送通知，实现恢复流程要求的重载接口。
func (s *BrandingStore) Reload() error {
	if err := s.store.Reload(); err != nil {
		return fmt.Errorf("failed to reload branding config: %w", err)
	}
	return nil
}
