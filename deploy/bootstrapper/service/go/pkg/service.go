package pkg

import (
	bscfg "github.com/getcouragenow/main/deploy/bootstrapper/service/go"
	"github.com/getcouragenow/main/deploy/bootstrapper/service/go/pkg/repo"
	bsrpc "github.com/getcouragenow/main/deploy/bootstrapper/service/go/rpc/v2"
	discoRepo "github.com/getcouragenow/mod/mod-disco/service/go/pkg/repo"
	"github.com/getcouragenow/protoc-gen-cobra/client"
	corebus "github.com/getcouragenow/sys-share/sys-core/service/go/pkg/bus"
	"github.com/getcouragenow/sys-share/sys-core/service/logging"
	accountRepo "github.com/getcouragenow/sys/sys-account/service/go/pkg/repo"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

type BootstrapService struct {
	proxyService *bsrpc.BSServiceService
	BsRepo       *repo.BootstrapRepo
}

func NewBootstrapService(cfg *bscfg.BootstrapConfig, l logging.Logger, accRepo *accountRepo.SysAccountRepo, discoRepo *discoRepo.ModDiscoRepo, cc grpc.ClientConnInterface, busClient *corebus.CoreBus) *BootstrapService {
	bsrepo := repo.NewBootstrapRepo(l, cfg.BSConfig.Domain, cfg.BSConfig.SavePath, accRepo, discoRepo, cc, busClient)
	svc := bsrpc.NewBSServiceService(bsrepo)
	return &BootstrapService{proxyService: svc, BsRepo: bsrepo}
}

func (b *BootstrapService) RegisterSvc(srv *grpc.Server) {
	bsrpc.RegisterBSServiceService(srv, b.proxyService)
}

func NewBootstrapCLI(bsRepo *repo.BootstrapRepo, option ...client.Option) *cobra.Command {
	rootCmd := bsrpc.BSServiceClientCommand(option...)
	cfg := client.NewConfig(option...)
	rootCmd.AddCommand(bsRepo.GenerateBSBypassCmd(cfg), bsRepo.ExecBSBypassCmd(cfg), bsRepo.NewBootstrapUpload(cfg))
	return rootCmd
}
