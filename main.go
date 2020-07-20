package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Logiraptor/concourse-blockers/deps"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/fly/rc"
	"github.com/concourse/concourse/go-concourse/concourse"
	"gopkg.in/gookit/color.v1"
)

type Edge struct {
	Resource string
	From, To string
	Trigger  bool
}

func main() {
	targetName := flag.String("t", "", "Concourse target, e.g. ci")
	pipelineName := flag.String("p", "", "Concourse pipeline, e.g. master")
	jobName := flag.String("j", "", "OPTIONAL, Concourse job, e.g. build")
	resourceName := flag.String("r", "", "OPTIONAL, Concourse resource, e.g. repo-name")
	flag.Parse()
	if *targetName == "" || *pipelineName == "" {
		flag.Usage()
		return
	}

	target, err := rc.LoadTarget(rc.TargetName(*targetName), false)
	if err != nil {
		log.Fatal(err)
	}
	client := target.Client()
	pipelines, err := client.ListPipelines()
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range pipelines {
		if p.Name == *pipelineName {
			ci := deps.NewCI(client, target.Team(), p)
			processPipeline(ci, target.Team(), p, *jobName, *resourceName)
			return
		}
	}
	log.Fatal("Could not find a pipeline with name:", *pipelineName)
}

func processPipeline(ci deps.CI, team concourse.Team, pipeline atc.Pipeline, jobName string, resourceName string) {
	jobs, err := team.ListJobs(pipeline.Name)
	if err != nil {
		log.Fatal(err)
	}

	jobsByName := make(map[string]atc.Job)
	for _, j := range jobs {
		jobsByName[j.Name] = j
	}

	for _, j := range jobs {
		if jobName != "" && jobName != j.Name {
			continue
		}
		fmt.Printf("%s\n", j.Name)

		prereqsByResource := ci.PrerequisitesForJob(j)

		for resource, jobs := range prereqsByResource {
			if resourceName != "" && resourceName != resource {
				continue
			}
			fmt.Printf("  %s\n", resource)

			fmt.Println("    The following jobs must pass for this trigger to occur:")
			fmt.Printf("    ")
			for _, path := range jobs {
				if path.HasNewInputs {
					color.Error.Printf("%s, ", path.Name)
				} else if triggersOnResource(path, resource) {
					color.Info.Printf("%s, ", path.Name)
				} else {
					color.Warn.Printf("%s, ", path.Name)
				}
			}
			fmt.Printf("%s\n", j.Name)

			fmt.Println()
		}
	}
}

func triggersOnResource(job atc.Job, resource string) bool {
	for _, r := range job.Inputs {
		if r.Name == resource {
			return r.Trigger
		}
	}
	return false
}

func clearPaths(jobsByName map[string]atc.Job, resource string, job atc.Job) [][]Edge {
	var results [][]Edge
	for _, input := range job.Inputs {
		if resource != input.Resource {
			continue
		}

		for _, p := range input.Passed {
			nextJob := jobsByName[p]

			edge := Edge{From: nextJob.Name, To: job.Name, Resource: input.Resource, Trigger: input.Trigger}

			subPaths := clearPaths(jobsByName, resource, nextJob)

			for _, subPath := range subPaths {
				results = append(results, append(subPath, edge))
			}
			if len(subPaths) == 0 {
				results = append(results, []Edge{edge})
			}
		}
	}

	return results
}
