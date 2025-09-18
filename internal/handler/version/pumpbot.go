package version

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"solana-bot/internal/logic/version"
	"solana-bot/internal/svc"
	"solana-bot/internal/types"
)

func PumpBot(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PumpBotRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := version.NewPumpBot(r.Context(), svcCtx, r)
		resp, err := l.PumpBot(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
