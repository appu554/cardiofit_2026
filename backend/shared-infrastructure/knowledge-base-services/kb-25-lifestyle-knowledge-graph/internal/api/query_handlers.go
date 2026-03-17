package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (s *Server) getCausalChain(c *gin.Context) {
	target := c.Param("target")
	if target == "" {
		sendError(c, http.StatusBadRequest, "target parameter required", "MISSING_TARGET", nil)
		return
	}

	chains, err := s.chainTraversal.GetChainsToTarget(c.Request.Context(), target)
	if err != nil {
		s.logger.Error("chain traversal failed", zap.String("target", target), zap.Error(err))
		sendError(c, http.StatusInternalServerError, "chain traversal failed", "TRAVERSAL_ERROR", nil)
		return
	}

	sendSuccess(c, gin.H{
		"target":      target,
		"chain_count": len(chains),
		"chains":      chains,
	}, nil)
}
