package main

import (
	"context"
	"fmt"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type Project struct {
	id            int
	sshURLToRepo  string
	httpURLToRepo string
	group         *Group
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

		subGroups := true
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
				case errsChan <- fmt.Errorf("failed to fetch projects for group %v: %w", group.id, err):
				}
				return
			}

			for _, project := range projects {
				prepare := &Project{
					id:            project.ID,
					sshURLToRepo:  project.SSHURLToRepo,
					httpURLToRepo: project.HTTPURLToRepo,
					group:         group,
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
