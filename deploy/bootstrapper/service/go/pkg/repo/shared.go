package repo

import (
	"context"
	"encoding/base64"
	"fmt"
	sysCorePkg "github.com/getcouragenow/sys-share/sys-core/service/go/pkg"
	"io/ioutil"
	"time"

	"github.com/getcouragenow/sys-share/sys-core/service/fakehelper"

	"github.com/getcouragenow/main/deploy/bootstrapper/service/go/pkg/fakedata"
	bsrpc "github.com/getcouragenow/main/deploy/bootstrapper/service/go/rpc/v2"
	accountPkg "github.com/getcouragenow/sys-share/sys-account/service/go/pkg"
	accountRepo "github.com/getcouragenow/sys/sys-account/service/go/pkg/repo"
)

const (
	defaultTimeout = 5 * time.Second
)

func (b *BootstrapRepo) sharedExecutor(ctx context.Context, bsAll *fakedata.BootstrapAll) (err error) {
	supers := bsAll.GetSuperUsers()
	orgs := bsAll.GetOrgs()
	projects := bsAll.GetProjects()
	regs := bsAll.GetRegularUsers()
	if b.accRepo != nil && b.discoRepo != nil {
		return b.sharedExecv2(ctx, supers, orgs, projects, regs)
	}
	if b.accClient != nil && b.discoClient != nil {
		return b.sharedExecv3(ctx, supers, orgs, projects, regs)
	}
	return fmt.Errorf("invalid argument, no repo or client defined for bootstrap")
}

func (b *BootstrapRepo) sharedExecv3(ctx context.Context, supers []*bsrpc.BSAccount, orgs []*bsrpc.BSOrg, projects []*bsrpc.BSProject, regularUsers []*bsrpc.BSRegularAccount) error {
	var err error
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err = b.resetAll(ctx)
	if err != nil {
		return err
	}
	for _, supe := range supers {
		supeRequest := &accountPkg.AccountNewRequest{
			Email:    supe.GetInitialSuperuser().GetEmail(),
			Password: supe.GetInitialSuperuser().GetPassword(),
			Roles: []*accountPkg.UserRoles{
				{
					Role:      accountPkg.SUPERADMIN,
					OrgID:     "",
					ProjectID: "",
				},
			},
		}
		avatar, err := fakehelper.GenFakeLogoBytes(128)
		if err != nil {
			return err
		}
		supeRequest.AvatarUploadBytes = base64.RawStdEncoding.EncodeToString(avatar)
		if _, err = b.accClient.NewAccount(ctx, supeRequest); err != nil {
			return err
		}
	}

	for _, org := range orgs {
		if _, err = b.accClient.NewOrg(ctx, accountPkg.OrgRequestFromProto(org.GetOrg())); err != nil {
			return err
		}
	}
	for _, proj := range projects {
		if _, err = b.accClient.NewProject(ctx, accountPkg.ProjectRequestFromProto(proj.GetProject())); err != nil {
			return err
		}
		if _, err = b.discoClient.NewDiscoProject(ctx, proj.GetProjectDetails()); err != nil {
			return err
		}
		if proj.GetSurveySchema() != nil {
			if _, err = b.discoClient.NewSurveyProject(ctx, proj.GetSurveySchema()); err != nil {
				return err
			}
		}
	}

	for _, reg := range regularUsers {
		acc, err := b.accClient.NewAccount(ctx, accountPkg.AccountNewRequestFromProto(reg.GetNewAccounts()))
		if err != nil {
			return err
		}
		updRequest := &accountPkg.AccountUpdateRequest{
			Id:       acc.GetId(),
			Verified: true,
		}
		if _, err = b.accClient.UpdateAccount(ctx, updRequest); err != nil {
			return err
		}
		if _, err = b.discoClient.NewSurveyUser(ctx, reg.GetSurveyValue()); err != nil {
			return err
		}
	}
	return nil
}

func (b *BootstrapRepo) sharedExecv2(ctx context.Context, supers []*bsrpc.BSAccount, orgs []*bsrpc.BSOrg, projects []*bsrpc.BSProject, regularAccounts []*bsrpc.BSRegularAccount) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	var err error
	err = b.resetAll(ctx)
	if err != nil {
		return err
	}
	for _, supe := range supers {
		superReq := &accountRepo.SuperAccountRequest{
			Email:    supe.InitialSuperuser.GetEmail(),
			Password: supe.GetInitialSuperuser().GetPassword(),
		}
		avatar, err := fakehelper.GenFakeLogoBytes(128)
		if err != nil {
			return err
		}
		superReq.AvatarBytes = base64.RawStdEncoding.EncodeToString(avatar)
		if err = b.accRepo.InitSuperUser(superReq); err != nil {
			return err
		}
	}

	for _, org := range orgs {
		if _, err = b.accRepo.NewOrg(ctx, accountPkg.OrgRequestFromProto(org.GetOrg())); err != nil {
			return err
		}
	}
	for _, proj := range projects {
		if _, err = b.accRepo.NewProject(ctx, accountPkg.ProjectRequestFromProto(proj.GetProject())); err != nil {
			return err
		}
		if _, err = b.discoRepo.NewDiscoProject(ctx, proj.GetProjectDetails()); err != nil {
			return err
		}
		if proj.GetSurveySchema() != nil {
			if _, err = b.discoRepo.NewSurveyProject(ctx, proj.GetSurveySchema()); err != nil {
				return err
			}
		}
	}
	for _, reg := range regularAccounts {
		var acc *accountPkg.Account
		acc, err = b.accRepo.NewAccount(ctx, accountPkg.AccountNewRequestFromProto(reg.GetNewAccounts()))
		if err != nil {
			return err
		}
		updRequest := &accountPkg.AccountUpdateRequest{
			Id:       acc.GetId(),
			Verified: true,
		}
		if _, err = b.accRepo.UpdateAccount(ctx, updRequest); err != nil {
			return err
		}
		if _, err = b.discoRepo.NewSurveyUser(ctx, reg.GetSurveyValue()); err != nil {
			return err
		}
	}
	return nil
}

func (b *BootstrapRepo) sharedGenBS(bsAll *fakedata.BootstrapAll, joined, extension string) (string, error) {
	switch extension {
	case "json":
		jbytes, err := bsAll.MarshalPretty()
		if err != nil {
			return "", err
		}
		return joined, ioutil.WriteFile(joined, jbytes, 0644)
	case "yml", "yaml":
		ybytes, err := bsAll.MarshalYaml()
		if err != nil {
			return "", err
		}
		return joined, ioutil.WriteFile(joined, ybytes, 0644)
	default:
		return "", fmt.Errorf("invalid filename extension: %s", extension)
	}
}

func (b *BootstrapRepo) resetAll(ctx context.Context) error {
	var err error
	_, err = b.busClient.Broadcast(ctx, &sysCorePkg.EventRequest{
		EventName:   "onResetAllModDisco",
		Initiator:   "bootstrap-service",
		UserId:      "",
		JsonPayload: []byte{},
	})
	if err != nil {
		return err
	}
	_, err = b.busClient.Broadcast(ctx, &sysCorePkg.EventRequest{
		EventName:   "onResetAllSysAccount",
		Initiator:   "bootstrap-service",
		UserId:      "",
		JsonPayload: []byte{},
	})
	if err != nil {
		return err
	}
	return nil
}
