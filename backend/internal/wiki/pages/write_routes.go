// 本文件将取消收藏请求映射为当前认证用户的收藏删除操作，并统一处理业务错误。
package pages

import (
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"

	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
)

// handleRemoveFavorite 仅删除当前认证用户与路由页面的收藏关联，不读取页面内容。
// 用户缺失时沿认证辅助函数的处理结果结束，存储错误交给统一页面错误响应。
func (r *Routes) handleRemoveFavorite(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	user := authmw.MustGetUser(c)
	if user == nil {
		return
	}
	if err := r.removeFavorite.Execute(c.Request.Context(), RemoveFavoriteInput{
		UserID: user.ID, PageID: id,
	}); err != nil {
		respondWithPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page unfavorited"})
}
