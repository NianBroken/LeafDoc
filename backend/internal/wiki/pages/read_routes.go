package pages

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/NianBroken/LeafDoc/internal/http/dto"
)

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

func (r *Routes) handleGetPage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	page, err := r.queries.GetPage(id)
	if err != nil {
		respondWithPageError(c, err)
		return
	}
	r.respondPage(c, http.StatusOK, page)
}
