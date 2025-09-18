package version

import (
	"net/http"

	"context"

	"solana-bot/internal/monitor"
	"solana-bot/internal/svc"
	"solana-bot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PumpBot struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewPumpBot(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *PumpBot {
	return &PumpBot{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *PumpBot) PumpBot(req *types.PumpBotRequest) (resp *types.PumpBotResponse, err error) {

	if req.Switch == "off" {
		monitor.PumpMonitor.Stop()
		return
	}
	p, err := monitor.NewPumpFunMonitor()
	if err != nil {
		logx.Error(err)
		return
	}
	monitor.PumpMonitor = p
	go p.Start()

	return
}
