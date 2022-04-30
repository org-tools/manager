package orgmanager

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v44/github"
)

type GitHub struct {
	client *github.Client
	config *githubConfig
}

type githubConfig struct {
	PEM string
	Org string
}

func (g *GitHub) InitFormUnmarshaler(unmarshaler func(any) error) (Target, error) {
	err := unmarshaler(&g.config)
	if err != nil {
		return nil, err
	}
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, 196252, 25342301, "github.pem")
	if err != nil {
		return nil, err
	}

	g.client = github.NewClient(&http.Client{Transport: itr})
	return g, nil
}

func (g GitHub) RootDepartment() UnionDepartment {
	return &githubTeam{
		client: g.client,
		org:    g.config.Org,
	}
}

type githubUser struct {
	client *github.Client
	raw    *github.User
}

type githubTeam struct {
	client *github.Client
	raw    *github.Team
	org    string
}

func (t githubTeam) Name() (name string) {
	//handle root dept as org
	if t.raw == nil {
		org, _, _ := t.client.Organizations.Get(context.Background(), t.org)
		return *org.Name
	}
	return *t.raw.Name
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
		teams, resp, _ = t.client.Teams.ListTeams(context.Background(), t.org, opts)
		firstDepthTeams := make([]*github.Team, 0)
		for _, team := range teams {
			if team.Parent == nil {
				firstDepthTeams = append(firstDepthTeams, team)
			}
		}
		teams = firstDepthTeams
	} else {
		teams, resp, _ = t.client.Teams.ListChildTeamsByParentSlug(context.Background(), *&t.org, *t.raw.Slug, opts)
	}
	for _, team := range teams {
		departments = append(departments, &githubTeam{
			client: t.client,
			raw:    team,
			org:    t.org,
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
	githubUsers, resp, _ := t.client.Teams.ListTeamMembersBySlug(context.Background(), t.org, *t.raw.Slug, opts)
	for _, user := range githubUsers {
		users = append(users, &githubUser{
			client: t.client,
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
	return *u.raw.NodeID
}

func (u githubUser) UserName() (name string) {
	if u.raw.Name != nil {
		fmt.Println(*u.raw.Name, *u.raw.Login)
		return *u.raw.Name
	}
	return *u.raw.Login
}
