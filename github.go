package orgmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v44/github"
)

type GitHub struct {
	client *github.Client
	config *githubConfig
}

func (g GitHub) GetTargetSlug() string {
	return g.config.Slug
}

func (g GitHub) GetPlatform() string {
	return g.config.Platform
}

type githubConfig struct {
	Platform       string
	Slug           string
	PEM            string
	Org            string
	OrgID          int64
	AppID          int64
	InstallationID int64
}

func (g *GitHub) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	err := unmarshaler(&g.config)
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.New(http.DefaultTransport, g.config.AppID, g.config.InstallationID, []byte(g.config.PEM))
	if err != nil {
		return nil, err
	}

	g.client = github.NewClient(&http.Client{Transport: itr})
	if g.config.OrgID == 0 {
		org, _, err := g.client.Organizations.Get(context.Background(), g.config.Org)
		if err != nil {
			return nil, err
		}
		g.config.OrgID = *org.ID
		fmt.Println(g.config.OrgID)
	}
	return g, nil
}

func (g *GitHub) RootDepartment() UnionDepartment {
	return &githubTeam{
		target: g,
	}
}

func (g *GitHub) LookupEntryDepartmentByInternalExternalIdentity(internalExtID ExternalIdentity) (UnionDepartment, error) {
	teamID, err := strconv.ParseInt(internalExtID.GetEntryID(), 10, 64)
	if err != nil {
		return nil, err
	}
	team, _, err := g.client.Teams.GetTeamByID(context.Background(), g.config.OrgID, teamID)
	if err != nil {
		return nil, err
	}
	return &githubTeam{target: g, raw: team}, nil
}

func (g *GitHub) LookupEntryUserByInternalExternalIdentity(internalExtID ExternalIdentity) (UnionUser, error) {
	return g.lookupGitHubUserByInternalExternalIdentity(internalExtID)
}

func (g *GitHub) lookupGitHubUserByInternalExternalIdentity(internalExtID ExternalIdentity) (*githubUser, error) {
	userID, err := strconv.ParseInt(internalExtID.GetEntryID(), 10, 64)
	if err != nil {
		return nil, err
	}
	user, _, err := g.client.Users.GetByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	return &githubUser{target: g, raw: user}, nil
}

type githubUser struct {
	target *GitHub
	raw    *github.User
}

type githubTeam struct {
	target *GitHub
	raw    *github.Team
}

func (t githubTeam) Name() (name string) {
	//handle root dept as org
	if t.raw == nil {
		org, _, _ := t.target.client.Organizations.Get(context.Background(), t.target.config.Org)
		return *org.Name
	}
	return *t.raw.Name
}

func (t githubTeam) DepartmentID() (departmentId string) {
	//handle root dept id as 0
	if t.raw == nil {
		return "0"
	}
	return strconv.FormatInt(*t.raw.ID, 10)
}

type githubTeamAddUserOptions struct {
	opts *github.TeamAddTeamMembershipOptions
}

func (o *githubTeamAddUserOptions) FromUnion(opts DepartmentModifyUserOptions) error {
	githubMembership, ok := map[DepartmentUserRole]string{
		DepartmentUserRoleMember: "member",
		DepartmentUserRoleAdmin:  "maintainer",
	}[opts.Role]
	if !ok {
		return errors.New("Role Mapping not found")
	}
	o.opts = &github.TeamAddTeamMembershipOptions{Role: githubMembership}
	return nil
}

func (t githubTeam) AddToDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error {
	if extID.GetTargetSlug() != t.target.config.Slug && extID.GetPlatform() != t.target.GetPlatform() {
		return errors.New("cannot add external user")
	}
	user, err := t.target.lookupGitHubUserByInternalExternalIdentity(extID)
	if err != nil {
		return fmt.Errorf("error finding user %s: %s", extID, err)
	}
	opts := new(githubTeamAddUserOptions)
	if err := opts.FromUnion(options); err != nil {
		return err
	}
	_, _, err = t.target.client.Teams.AddTeamMembershipBySlug(context.Background(), t.target.config.Org, *t.raw.Slug,
		*user.raw.Login, opts.opts)
	return err
}

func (t githubTeam) DeleteFromDepartment(options DepartmentModifyUserOptions, extID ExternalIdentity) error {
	if extID.GetTargetSlug() != t.target.config.Slug && extID.GetPlatform() != t.target.GetPlatform() {
		return errors.New("cannot delete external user")
	}
	user, err := t.target.lookupGitHubUserByInternalExternalIdentity(extID)
	if err != nil {
		return fmt.Errorf("error finding user %s: %s", extID, err)
	}
	opts := new(githubTeamAddUserOptions)
	if err := opts.FromUnion(options); err != nil {
		return err
	}
	_, err = t.target.client.Teams.RemoveTeamMembershipBySlug(context.Background(), t.target.config.Org, *t.raw.Slug,
		*user.raw.Login)
	return err
}

func (t githubTeam) CreateSubDepartment(options DepartmentCreateOptions) (UnionDepartment, error) {
	team, _, err := t.target.client.Teams.CreateTeam(context.Background(), t.target.config.Org, github.NewTeam{
		Name:         options.Name,
		Description:  &options.Description,
		ParentTeamID: t.raw.ID,
	})
	return &githubTeam{
		target: t.target,
		raw:    team,
	}, err
}

func (t githubTeam) SubDepartments() (departments []UnionDepartment) {
	opts := &github.ListOptions{
		Page:    0,
		PerPage: 100,
	}
	var (
		teams []*github.Team
		resp  *github.Response
	)
FETCH_TEAMS:
	if t.raw == nil {
		teams, resp, _ = t.target.client.Teams.ListTeams(context.Background(), t.target.config.Org, opts)
		firstDepthTeams := make([]*github.Team, 0)
		for _, team := range teams {
			if team.Parent == nil {
				firstDepthTeams = append(firstDepthTeams, team)
			}
		}
		teams = firstDepthTeams
	} else {
		teams, resp, _ = t.target.client.Teams.ListChildTeamsByParentSlug(context.Background(), t.target.config.Org, *t.raw.Slug, opts)
	}
	for _, team := range teams {
		departments = append(departments, &githubTeam{
			target: t.target,
			raw:    team,
		})
	}

	if resp.NextPage != resp.LastPage {
		opts.Page = resp.NextPage
		goto FETCH_TEAMS
	}
	return departments
}

func (t githubTeam) Users() (users []UnionUser) {
	if t.raw == nil {
		return users
	}
	opts := &github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: 100,
		},
	}
FETCH_USERS:
	githubUsers, resp, _ := t.target.client.Teams.ListTeamMembersBySlug(context.Background(), t.target.config.Org, *t.raw.Slug, opts)
	for _, user := range githubUsers {
		users = append(users, &githubUser{
			target: t.target,
			raw:    user,
		})
	}
	if resp.NextPage != resp.LastPage {
		opts.ListOptions.Page = resp.NextPage
		goto FETCH_USERS
	}
	return users
}

func (u githubUser) UserId() (userId string) {
	return strconv.FormatInt(*u.raw.ID, 10)
}

func (u githubUser) UserName() (name string) {
	if u.raw.Name != nil {
		fmt.Println(*u.raw.Name, *u.raw.Login)
		return *u.raw.Name
	}
	return *u.raw.Login
}

func (u githubUser) GetEmailSet() []string {
	return []string{*u.raw.Email}
}
