package main

import (
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/go-concourse/concourse"
)

type Graph []Edge

type Edge struct {
	Resource string
	From, To string
	Trigger  bool
}

func findDependencies(client concourse.Client, pipeline atc.Pipeline, jobName string) (map[string]Graph, error) {
	jobs, err := client.Team("main").ListJobs(pipeline.Name)
	if err != nil {
		return nil, err
	}
	var jobsByName = make(map[string]atc.Job)
	for _, job := range jobs {
		jobsByName[job.Name] = job
	}

	var graphsByResource = make(map[string]Graph)
	for _, input := range jobsByName[jobName].Inputs {

		var seen = make(map[Edge]struct{})
		recurse(jobsByName, jobName, input.Resource, func(e Edge) bool {
			_, visited := seen[e]
			if !visited {

				seen[e] = struct{}{}
				graphsByResource[e.Resource] = append(graphsByResource[e.Resource], e)

				return true
			}
			return false
		})

	}

	for _, graph := range graphsByResource {
		reverse(graph)
	}
	return graphsByResource, nil
}

func reverse(edges []Edge) {
	for i, j := 0, len(edges)-1; i < j; i, j = i+1, j-1 {
		edges[i], edges[j] = edges[j], edges[i]
	}
}

func recurse(jobs map[string]atc.Job, jobName, resourceName string, callback func(Edge) bool) {
	job := jobs[jobName]

	for _, input := range job.Inputs {
		if input.Resource != resourceName {
			continue
		}
		for _, j := range input.Passed {
			if callback(Edge{
				From:     j,
				To:       jobName,
				Resource: input.Resource,
				Trigger:  input.Trigger,
			}) {
				recurse(jobs, j, resourceName, callback)
			}
		}
	}
}
