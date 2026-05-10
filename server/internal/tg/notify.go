package tg

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/easyspace-ai/polybet/internal/config"
)

const maxTelegramMessageRunes = 3500

// Notify sends a plain-text Telegram message if bot token and chat id are configured.
// It is best-effort, non-blocking, and ignores empty text.
func Notify(cfg *config.Config, log *slog.Logger, text string) {
	if cfg == nil {
		return
	}
	if log == nil {
		log = slog.Default()
	}
	token := strings.TrimSpace(cfg.TelegramBotToken)
	chat := strings.TrimSpace(cfg.TelegramChatID)
	t := strings.TrimSpace(text)
	if token == "" || chat == "" || t == "" {
		return
	}
	if len([]rune(t)) > maxTelegramMessageRunes {
		rs := []rune(t)
		t = string(rs[:maxTelegramMessageRunes]) + "…"
	}
	go func(msg string) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		form := url.Values{}
		form.Set("chat_id", chat)
		form.Set("text", msg)
		u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", url.PathEscape(token))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
		if err != nil {
			log.Debug("telegram_send_skip", "err", err.Error())
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		hc := &http.Client{Timeout: 14 * time.Second}
		resp, err := hc.Do(req)
		if err != nil {
			log.Warn("telegram_send_failed", "err", err.Error())
			return
		}
		resp.Body.Close()
	}(t)
}
