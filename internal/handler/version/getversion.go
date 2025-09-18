package version

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"solana-bot/internal/logic/version"
	"solana-bot/internal/svc"
	"solana-bot/internal/types"
)

func GetVersion(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetVersionRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := version.NewGetVersion(r.Context(), svcCtx)
		resp, err := l.GetVersion(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
