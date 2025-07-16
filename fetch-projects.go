package main

import (
	"context"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type Project struct {
	id                int
	sshURLToRepo      string
	httpURLToRepo     string
	path              string
	pathWithNamespace string
	group             *Group
}

func fetchProjectByGroup(ctx context.Context, client ProjectsService, group *Group) (<-chan *Project, <-chan error) {
	dataChan := make(chan *Project)
	errsChan := make(chan error)

	go func() {
		defer func() {
			close(dataChan)
			close(errsChan)
		}()

		if group == nil {
			select {
			case <-ctx.Done():
			case errsChan <- ErrorNoGroupPassed:
			}
			return
		}

		subGroups := false
		withShared := false
		simple := true
		opt := &gitlab.ListGroupProjectsOptions{
			IncludeSubGroups: &subGroups,
			WithShared:       &withShared,
			Simple:           &simple,
		}
		opt.PerPage = 100

		for {
			projects, resp, err := client.ListGroupProjects(group.id, opt, gitlab.WithContext(ctx))
			if err != nil {
				select {
				case <-ctx.Done():
				case errsChan <- &ErrorProjectsFetching{group.id, err}:
				}
				return
			}

			for _, project := range projects {
				if project == nil {
					continue
				}

				prepare := &Project{
					id:                project.ID,
					path:              project.Path,
					pathWithNamespace: project.PathWithNamespace,
					sshURLToRepo:      project.SSHURLToRepo,
					httpURLToRepo:     project.HTTPURLToRepo,
					group:             group,
				}

				select {
				case <-ctx.Done():
					return
				case dataChan <- prepare:
				}
			}

			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}()

	return dataChan, errsChan
}
