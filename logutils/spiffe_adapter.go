package logutils

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/spiffe/go-spiffe/v2/logger"
)

var _ logger.Logger = (*SPIFFEAdapter)(nil)

type SPIFFEAdapter struct {
	ctx    context.Context
	logger *slog.Logger
}

func NewSPIFFEAdapter(ctx context.Context, logger *slog.Logger) *SPIFFEAdapter {
	if ctx == nil {
		ctx = context.Background()
	}
	return &SPIFFEAdapter{
		ctx:    ctx,
		logger: logger,
	}
}

func (s *SPIFFEAdapter) newRecord(level slog.Level, format string, rest ...any) slog.Record {
	var pc [1]uintptr
	// 3 - Callers, newRecord, Debugf
	runtime.Callers(3, pc[:])
	return slog.NewRecord(time.Now(), level, fmt.Sprintf(format, rest...), pc[0])
}

func (s *SPIFFEAdapter) handle(rec slog.Record) {
	_ = s.logger.Handler().Handle(s.ctx, rec)
}

func (s *SPIFFEAdapter) Debugf(format string, rest ...any) {
	if !s.logger.Enabled(s.ctx, slog.LevelDebug) {
		return
	}

	rec := s.newRecord(slog.LevelDebug, format, rest...)
	s.handle(rec)
}

func (s *SPIFFEAdapter) Infof(format string, rest ...any) {
	if !s.logger.Enabled(s.ctx, slog.LevelInfo) {
		return
	}

	rec := s.newRecord(slog.LevelInfo, format, rest...)
	s.handle(rec)
}

func (s *SPIFFEAdapter) Warnf(format string, rest ...any) {
	if !s.logger.Enabled(s.ctx, slog.LevelWarn) {
		return
	}

	rec := s.newRecord(slog.LevelWarn, format, rest...)
	s.handle(rec)
}

func (s *SPIFFEAdapter) Errorf(format string, rest ...any) {
	if !s.logger.Enabled(s.ctx, slog.LevelError) {
		return
	}

	rec := s.newRecord(slog.LevelError, format, rest...)
	s.handle(rec)
}
