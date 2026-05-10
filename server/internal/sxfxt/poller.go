// Package sxfxt polls SX Bet HTTP APIs for fixture status and live scores (read-only).
// Real-time Centrifugo streaming is not wired here yet; REST polling feeds fixturecache + WS.
package sxfxt

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/easyspace-ai/polybet/internal/config"
	"github.com/easyspace-ai/polybet/internal/fixturecache"
	"github.com/easyspace-ai/polybet/internal/store"
	"github.com/easyspace-ai/polybet/internal/wsrelay"
)

const batchSize = 30

// Run polls SX REST for fixture status + live scores (read-only, no on-chain SX).
func Run(ctx context.Context, cfg *config.Config, st *store.Store, hub *wsrelay.Hub, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	if strings.TrimSpace(cfg.SXBetAPIKey) == "" {
		log.Info("sx_fixture_poll_disabled", "reason", "SX_BET_API_KEY empty")
		return
	}
	hc := &http.Client{Timeout: 30 * time.Second}
	pollAll := func() {
		ids, err := st.ListDistinctSxEventIDs(context.Background())
		if err != nil || len(ids) == 0 {
			return
		}
		for _, batch := range chunkStrings(ids, batchSize) {
			pollBatch(context.Background(), hc, cfg, batch, hub, log)
		}
	}
	pollAll()
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pollAll()
		}
	}
}

func chunkStrings(xs []string, n int) [][]string {
	var out [][]string
	for i := 0; i < len(xs); i += n {
		j := i + n
		if j > len(xs) {
			j = len(xs)
		}
		out = append(out, xs[i:j])
	}
	return out
}

func pollBatch(ctx context.Context, hc *http.Client, cfg *config.Config, ids []string, hub *wsrelay.Hub, log *slog.Logger) {
	q := strings.Join(ids, ",")
	// Status map
	u := strings.TrimRight(cfg.SXBetAPIURL, "/") + "/fixture/status?sportXEventIds=" + q
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("x-api-key", cfg.SXBetAPIKey)
	resp, err := hc.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		var body struct {
			Data map[string]struct {
				Status int `json:"status"`
			} `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		now := time.Now().UnixMilli()
		for sxID, row := range body.Data {
			prev, _ := fixturecache.Global.Get(sxID)
			st := mergeFixture(prev, sxID, row.Status, now)
			fixturecache.Global.Put(sxID, st)
			if hub != nil {
				hub.BroadcastJSON(map[string]any{"type": "fixtureUpdate", "data": st})
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}
	// Live scores
	u2 := strings.TrimRight(cfg.SXBetAPIURL, "/") + "/live-scores?sportXEventIds=" + q
	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, u2, nil)
	req2.Header.Set("x-api-key", cfg.SXBetAPIKey)
	resp2, err := hc.Do(req2)
	if err != nil || resp2.StatusCode != http.StatusOK {
		if resp2 != nil {
			resp2.Body.Close()
		}
		return
	}
	defer resp2.Body.Close()
	var ls struct {
		Data []struct {
			SportXeventID string `json:"sportXeventId"`
			TeamOneScore  int    `json:"teamOneScore"`
			TeamTwoScore  int    `json:"teamTwoScore"`
			CurrentPeriod string `json:"currentPeriod"`
			PeriodTime    string `json:"periodTime"`
			Periods       []any  `json:"periods"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&ls); err != nil {
		log.Warn("sx_live_scores_json", "err", err)
		return
	}
	now := time.Now().UnixMilli()
	for _, row := range ls.Data {
		if row.SportXeventID == "" {
			continue
		}
		prev, _ := fixturecache.Global.Get(row.SportXeventID)
		st := mergeLive(prev, row.SportXeventID, row.TeamOneScore, row.TeamTwoScore,
			row.CurrentPeriod, row.PeriodTime, row.Periods, now)
		fixturecache.Global.Put(row.SportXeventID, st)
		if hub != nil {
			hub.BroadcastJSON(map[string]any{"type": "fixtureUpdate", "data": st})
		}
	}
}

func mergeFixture(prev map[string]any, sxID string, status int, updatedAt int64) map[string]any {
	st := map[string]any{
		"sxEventId": sxID, "status": status,
		"teamOneScore": 0, "teamTwoScore": 0, "currentPeriod": "", "periodTime": "-1",
		"periods": []any{}, "updatedAt": updatedAt,
	}
	if prev != nil {
		if v, ok := prev["teamOneScore"].(float64); ok {
			st["teamOneScore"] = int(v)
		} else if v, ok := prev["teamOneScore"].(int); ok {
			st["teamOneScore"] = v
		}
		if v, ok := prev["teamTwoScore"].(float64); ok {
			st["teamTwoScore"] = int(v)
		} else if v, ok := prev["teamTwoScore"].(int); ok {
			st["teamTwoScore"] = v
		}
		if s, ok := prev["currentPeriod"].(string); ok {
			st["currentPeriod"] = s
		}
		if s, ok := prev["periodTime"].(string); ok {
			st["periodTime"] = s
		}
		if p, ok := prev["periods"].([]any); ok {
			st["periods"] = p
		}
	}
	return st
}

func mergeLive(prev map[string]any, sxID string, t1, t2 int, period, ptime string, periods []any, updatedAt int64) map[string]any {
	status := 2
	if prev != nil {
		if v, ok := prev["status"].(float64); ok {
			status = int(v)
		} else if v, ok := prev["status"].(int); ok {
			status = v
		}
	}
	st := map[string]any{
		"sxEventId": sxID, "status": status,
		"teamOneScore": t1, "teamTwoScore": t2,
		"currentPeriod": period, "periodTime": ptime,
		"periods": periods, "updatedAt": updatedAt,
	}
	if len(periods) == 0 && prev != nil {
		if p, ok := prev["periods"].([]any); ok {
			st["periods"] = p
		}
	}
	return st
}
