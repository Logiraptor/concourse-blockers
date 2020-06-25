package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/fly/rc"
	"github.com/concourse/concourse/go-concourse/concourse"
)

func main() {
	targetName := flag.String("t", "cp", "Concourse target, e.g. ci")
	pipelineName := flag.String("p", "master", "Concourse pipeline, e.g. master")
	flag.Parse()
	// if *targetName == "" || *pipelineName == "" {
	// 	flag.Usage()
	// 	return
	// }

	target, err := rc.LoadTarget(rc.TargetName(*targetName), false)
	if err != nil {
		log.Fatal(err)
	}
	client := target.Client()
	pipeline, _, err := client.Team("main").Pipeline(*pipelineName)
	if err != nil {
		log.Fatal(err)
	}

	// http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
	output := processPipeline(client, pipeline)
	tmpl := template.Must(template.New("root").ParseFiles("index.html"))
	tmpl.ExecuteTemplate(ioutil.Discard, "index.html", output)
	// })
	// fmt.Println("Listening on port", os.Getenv("PORT"))
	// http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

type cell struct {
	Job    string
	Status string
}

type cellRow struct {
	Version string
	Cells   []cell
}

type result struct {
	Resource string
	CellRows []cellRow
}

func processPipeline(client concourse.Client, pipeline atc.Pipeline) []result {
	var start = time.Now()
	var tick = func(name string) {
		fmt.Println(name, time.Since(start))
		start = time.Now()
	}
	tick("start")
	team := client.Team(pipeline.TeamName)
	deps, err := findDependencies(client, pipeline, "promote_trigger")
	if err != nil {
		log.Fatal(err)
	}
	tick("findDependencies")

	config, _, _, err := team.PipelineConfig(pipeline.Name)
	tick("pipelineConfig")

	var output []result
	for resourceName, edges := range deps {
		resourceConfig, _ := config.Resources.Lookup(resourceName)

		versions, _, _, err := team.ResourceVersions(pipeline.Name, resourceName, concourse.Page{Limit: 1}, nil)
		if err != nil {
			log.Fatal(err)
		}
		latestVersion := versions[0]

		builds, _, err := team.BuildsWithVersionAsInput(pipeline.Name, resourceName, latestVersion.ID)
		if err != nil {
			log.Fatal(err)
		}

		cells := findLatestAndPendingJobs(edges, builds)

		output = append(output, result{
			Resource: resourceName,
			CellRows: []cellRow{
				{Version: extractVersion(resourceConfig, latestVersion), Cells: cells},
			},
		})
		tick(resourceName)
	}

	sort.Slice(output, func(i, j int) bool {
		return output[i].Resource < output[j].Resource
	})
	tick("sort")

	return output
}

func extractVersion(config atc.ResourceConfig, version atc.ResourceVersion) string {
	switch config.Type {
	case "iam-s3-resource":
		versionRegexp := regexp.MustCompile(config.Source["regexp"].(string))
		matchedVersion := versionRegexp.FindStringSubmatch(version.Version["path"])
		return matchedVersion[1]
	case "iam-registry-image":
		return version.Version["digest"]
	default:
		fmt.Println("Teach me how to find a version for ", config.Type, version.Version)
		return strconv.Itoa(version.ID)
	}
}

func findLatestAndPendingJobs(expectedJobs []Edge, actualBuilds []atc.Build) []cell {
	var cells []cell
	for _, edge := range expectedJobs {
		var latestBuild atc.Build
		latestBuild.Status = "missing"
		// for each edge, find latest build
		for _, build := range actualBuilds {
			if build.JobName != edge.From {
				continue
			}
			if build.EndTime > latestBuild.EndTime {
				latestBuild = build
			}
		}

		cells = append(cells, cell{
			Job:    edge.From,
			Status: latestBuild.Status,
		})
	}
	return cells
}
