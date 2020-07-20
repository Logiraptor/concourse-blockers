package deps

import (
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/go-concourse/concourse"
)

func findDependencies(client concourse.Client, pipeline atc.Pipeline, jobName string) (map[string][]atc.Job, error) {
	jobs, err := client.Team("main").ListJobs(pipeline.Name)
	if err != nil {
		return nil, err
	}
	var jobsByName = make(map[string]atc.Job)
	for _, job := range jobs {
		jobsByName[job.Name] = job
	}

	var graphsByResource = make(map[string][]atc.Job)
	for _, input := range jobsByName[jobName].Inputs {

		var graph = []atc.Job{}
		var seen = make(map[string]struct{})
		recurse(jobsByName, jobName, input.Resource, func(e atc.Job) bool {
			_, visited := seen[e.Name]
			if !visited {

				seen[e.Name] = struct{}{}
				graph = append(graph, e)

				return true
			}
			return false
		})
		graphsByResource[input.Resource] = graph

	}

	for _, graph := range graphsByResource {
		reverse(graph)
	}
	return graphsByResource, nil
}

func reverse(edges []atc.Job) {
	for i, j := 0, len(edges)-1; i < j; i, j = i+1, j-1 {
		edges[i], edges[j] = edges[j], edges[i]
	}
}

func recurse(jobs map[string]atc.Job, jobName, resourceName string, callback func(atc.Job) bool) {
	job := jobs[jobName]

	for _, input := range job.Inputs {
		if input.Resource != resourceName {
			continue
		}
		for _, j := range input.Passed {
			if callback(jobs[j]) {
				recurse(jobs, j, resourceName, callback)
			}
		}
	}
}
