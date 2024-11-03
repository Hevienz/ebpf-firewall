package server

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

func (s *Server) Ping(c fiber.Ctx) error {
	return c.SendString(fmt.Sprintf("pong %s", time.Now().Format(time.DateTime)))
}

func (s *Server) GetLinkType(c fiber.Ctx) error {
	return c.SendString(s.ebpf.GetLinkType())
}

func (s *Server) GetMetricsReport(c fiber.Ctx) error {
	topStr := c.Query("top", "10")
	top, err := strconv.Atoi(topStr)
	if err != nil {
		top = 10
	}
	return c.JSON(s.metrics.GenerateReport(top))
}

func (s *Server) GetSources(c fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("page_size", "20"))
	if err != nil {
		pageSize = 20
	}
	order := c.Query("order", "last_seen_at")
	sortDir := c.Query("sort", "desc")
	return c.JSON(s.metrics.GetSources(page, pageSize, order, sortDir))
}

func (s *Server) GetTargets(c fiber.Ctx) error {
	sourceId := c.Params("sourceId")
	if sourceId == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.Query("page_size", "20"))
	if err != nil {
		pageSize = 20
	}
	order := c.Query("order", "last_seen_at")
	sortDir := c.Query("sort", "desc")
	return c.JSON(s.metrics.GetTargets(sourceId, page, pageSize, order, sortDir))
}

func (s *Server) GetBlackList(c fiber.Ctx) error {

	return nil
}

func (s *Server) AddBlack(c fiber.Ctx) error {
	return nil
}

func (s *Server) DeleteBlack(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	return nil
}
