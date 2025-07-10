package main

import gitlab "gitlab.com/gitlab-org/api/client-go"

type GroupsService interface {
	GetGroup(gid string, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error)
	ListGroups(opt *gitlab.ListGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error)
}

type ProjectsService interface {
	ListGroupProjects(gid int, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error)
}

type Gitlab struct {
	client *gitlab.Client
}

func NewGitlab(client *gitlab.Client) *Gitlab {
	return &Gitlab{
		client: client,
	}
}

func (g *Gitlab) GetGroup(gid string, opt *gitlab.GetGroupOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Group, *gitlab.Response, error) {
	return g.client.Groups.GetGroup(gid, opt, options...)
}

func (g *Gitlab) ListGroups(opt *gitlab.ListGroupsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Group, *gitlab.Response, error) {
	return g.client.Groups.ListGroups(opt, options...)
}

func (g *Gitlab) ListGroupProjects(gid int, opt *gitlab.ListGroupProjectsOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Project, *gitlab.Response, error) {
	return g.client.Groups.ListGroupProjects(gid, opt, options...)
}
