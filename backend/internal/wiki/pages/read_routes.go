package pages

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/NianBroken/LeafDoc/internal/http/dto"
)

// handleGetTree 将当前根节点转成 API 树，未传 depth 时展开全部层级。
// 非空 depth 解析失败使用 -1，DTO 将负深度按不限制处理，零深度不展开子节点。
func (r *Routes) handleGetTree(c *gin.Context) {
	root := r.treeService.GetTree()
	depthStr := strings.TrimSpace(c.Query("depth"))
	if depthStr == "" {
		c.JSON(http.StatusOK, dto.ToAPINode(root, "", r.userResolver))
		return
	}
	depth, err := strconv.Atoi(depthStr)
	if err != nil {
		depth = -1
	}
	c.JSON(http.StatusOK, dto.ToAPINodeWithDepth(root, "", r.userResolver, depth))
}

// handleGetPage 去除路由标识两端空白，按标识查询页面并使用统一元信息响应入口。
func (r *Routes) handleGetPage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	page, err := r.queries.GetPage(id)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, page)
}
