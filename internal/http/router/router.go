package router

import (
	"io"
	"net/http"
	"os"
	"strings"

	votingv1 "vote-system/internal/gen/voting/v1"
	"vote-system/internal/http/handler"
	"vote-system/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(client votingv1.VotingServiceClient) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.TokenBucket(200, 400, 20, 40))

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	hub := handler.NewSSEHubForRouter()
	h := handler.New(client, hub)

	r.StaticFile("/", "./web/index.html")
	r.Static("/static", "./web/static")

	r.NoRoute(func(c *gin.Context) {
		// SPA fallback: return index.html for unknown GET routes.
		if c.Request.Method != http.MethodGet {
			c.Status(http.StatusNotFound)
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/polls") || strings.HasPrefix(c.Request.URL.Path, "/votes") || strings.HasPrefix(c.Request.URL.Path, "/users") || strings.HasPrefix(c.Request.URL.Path, "/events") {
			c.Status(http.StatusNotFound)
			return
		}
		f, err := os.Open("./web/index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer f.Close()
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, f)
	})

	api := r.Group("/")
	api.Use(middleware.RequireUser())
	{
		polls := api.Group("/polls")
		{
			polls.POST("/createPoll", h.CreatePoll)
			polls.GET("/:id", h.GetPoll)
			polls.GET("/close/:id", h.ClosePoll)
			polls.GET("/delete/:id", h.DeletePoll)
			polls.GET("/search", h.SearchPolls)
			polls.GET("/get_polls_by_vote", h.GetPollStats)
			polls.GET("/public", h.ListPublicPolls)
			polls.GET("/public/stats", h.ListPublicPollStats)
			polls.GET("/my_created/stats", h.ListMyCreatedPollStats)
		}

		votes := api.Group("/votes")
		{
			votes.POST("/:poll_id/vote", h.Vote)
			votes.DELETE("/:poll_id/vote", h.UndoVote)
		}

		users := api.Group("/users")
		{
			users.GET("/me/votes", h.GetMyVotes)
		}

		events := api.Group("/events")
		{
			events.GET("/results", h.ResultsSSE)
			events.GET("/polls/:poll_id", h.PollSSE)
		}
	}

	return r
}
