package pages

import (
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"

	authmw "github.com/perber/wiki/internal/http/middleware/auth"
)

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
