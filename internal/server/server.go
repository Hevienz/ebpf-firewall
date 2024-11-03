package server

import (
	"embed"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/danger-dream/ebpf-firewall/internal/config"
	"github.com/danger-dream/ebpf-firewall/internal/ebpf"
	"github.com/danger-dream/ebpf-firewall/internal/metrics"
	"github.com/danger-dream/ebpf-firewall/internal/processor"
	"github.com/danger-dream/ebpf-firewall/internal/server/middleware"
	"github.com/danger-dream/ebpf-firewall/internal/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

type Server struct {
	app       *fiber.App
	ebpf      *ebpf.EBPFManager
	metrics   *metrics.MetricsCollector
	processor *processor.Processor
	security  *middleware.Security
	limiter   *middleware.Limiter
}

func New(ebpf *ebpf.EBPFManager, metrics *metrics.MetricsCollector, processor *processor.Processor) *Server {
	app := fiber.New()
	server := &Server{
		app:       app,
		ebpf:      ebpf,
		metrics:   metrics,
		processor: processor,
	}
	server.initServer()
	server.setupRoutes()
	return server
}

func (s *Server) initServer() {
	config := config.GetConfig()
	s.security = middleware.NewSecurity(config.DataDir, config.Security.IPErrorThreshold, config.Security.ErrorWindow)
	s.limiter = middleware.NewLimiter(config.RateLimit.RateLimitRequest, config.RateLimit.RateLimitInterval)
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))
	s.app.Use(logger.New(logger.Config{
		TimeZone:   "Asia/Shanghai",
		TimeFormat: "2006-01-02 15:04:05",
		Next: func(c fiber.Ctx) bool {
			paths := []string{"/api/v1/metrics", "/api/v1/sources"}
			for _, path := range paths {
				if strings.HasPrefix(c.Path(), path) {
					return true
				}
			}
			return false
		},
	}))
}

func (s *Server) setupRoutes() {
	config := config.GetConfig()
	api := s.app.Group("/api/v1", func(c fiber.Ctx) error {
		// fiber ctx.IP() -> fasthttp ctx.RemoteIP()
		// Get IP address from header
		srcIP := c.Get("X-Real-IP")
		if srcIP == "" {
			forwardedFor := c.Get("X-Forwarded-For")
			if forwardedFor != "" {
				srcIP = strings.Split(forwardedFor, ",")[0]
			}
		}

		// If the IP address is not obtained from the header, use the IP address directly connected to the server
		if srcIP == "" {
			srcIP = c.IP()
		}
		// If the IP address is not local, check if it is blocked or rate limited
		if !utils.IsLocalIP(srcIP) {
			if s.security.IsBlocked(srcIP) {
				return c.SendStatus(fiber.StatusForbidden)
			}
			if s.limiter.IsRateLimited(srcIP) {
				return c.SendStatus(fiber.StatusTooManyRequests)
			}
		}
		if config.Auth == "" {
			return c.Next()
		}
		// Get auth from query or header
		auth := c.Get("Authorization")
		if auth == "" && c.Method() == "GET" {
			auth = c.Query("auth", "")
		}
		if auth != config.Auth {
			s.security.AddRecord(srcIP, "auth failed: "+auth)
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		return c.Next()
	})
	api.Get("/ping", s.Ping)
	api.Get("/link-type", s.GetLinkType)

	api.Get("/metrics", s.GetMetricsReport)
	api.Get("/sources", s.GetSources)
	api.Get("/:sourceId/targets", s.GetTargets)

	black := api.Group("/black")
	black.Get("/", s.GetBlackList)
	black.Post("/", s.AddBlack)
	black.Delete("/:id", s.DeleteBlack)

}

func (s *Server) ServeStaticDirectory(directory string) {
	s.app.Get("/*", func(c fiber.Ctx) error {
		path := c.Path()
		if path == "/" {
			path = "/index.html"
		}
		filePath := directory + path
		if stat, err := os.Stat(filePath); err != nil || stat.IsDir() {
			return c.SendStatus(fiber.StatusNotFound)
		}
		ext := filepath.Ext(filePath)
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			c.Set("Content-Type", mimeType)
		}
		return c.SendFile(filePath)
	})
}

// ServeEmbeddedFiles handles static file serving for embedded files since Fiber v3's static middleware
// cannot properly handle embed.FS. This implementation provides a custom solution for serving static
// files from the embedded filesystem, with proper mime type detection and index.html fallback for the
// root path.
func (s *Server) ServeEmbeddedFiles(staticFS embed.FS) {
	s.app.Get("/*", func(c fiber.Ctx) error {
		path := c.Path()
		log.Println("serving static file: ", path)
		if path == "/" {
			path = "/index.html"
		}
		filePath := filepath.Join("web/dist", path)
		content, err := staticFS.ReadFile(filePath)
		if err != nil {
			log.Println("static file not found: ", filePath)
			return c.SendStatus(fiber.StatusNotFound)
		}
		ext := filepath.Ext(filePath)
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			c.Set("Content-Type", mimeType)
		}
		return c.Send(content)
	})
}

func (s *Server) HandleStatusNotFound() {
	s.app.Use(func(c fiber.Ctx) error {
		s.security.AddRecord(c.IP(), "not found: "+c.Path())
		return c.SendStatus(fiber.StatusNotFound)
	})
}

func (s *Server) Start() error {
	return s.app.Listen(config.GetConfig().Addr)
}

func (s *Server) Close() error {
	return s.app.Shutdown()
}
