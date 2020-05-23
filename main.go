package main

import (
	"flag"
	"fmt"
	"log"

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
			processPipeline(client, p, *jobName, *resourceName)
			return
		}
	}
	log.Fatal("Could not find a pipeline with name:", *pipelineName)
}

func processPipeline(client concourse.Client, pipeline atc.Pipeline, jobName string, resourceName string) {
	team := client.Team(pipeline.TeamName)
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

		for _, r := range j.Inputs {
			if resourceName != "" && resourceName != r.Resource {
				continue
			}
			fmt.Printf("  %s\n", r.Resource)

			paths := clearPaths(jobsByName, r.Resource, j)

			if len(r.Passed) == 0 {
				color.Info.Printf("    No passed constraints\n")
				continue
			}

			fmt.Println("    The following jobs must pass for this trigger to occur:")
			for _, path := range paths {
				fmt.Printf("    ")
				for _, step := range path {
					if step.Trigger {
						color.Info.Printf("%s, ", step.From)
					} else {
						color.Error.Printf("%s, ", step.From)
					}
				}
				fmt.Printf("%s\n", j.Name)
			}

			fmt.Println()
		}
	}
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
