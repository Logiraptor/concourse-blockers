package deps

import (
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/go-concourse/concourse"
)

type CI struct {
	client   concourse.Client
	team     concourse.Team
	pipeline atc.Pipeline
	err      error
}

type BuildResult struct {
	Build    atc.Build
	Resource atc.Resource
	Version  atc.Version
}

func NewCI(client concourse.Client, team concourse.Team, pipeline atc.Pipeline) CI {
	return CI{
		client:   client,
		team:     team,
		pipeline: pipeline,
	}
}

func (c CI) ResourcesForJob(job atc.Job) []atc.Resource {
	result := []atc.Resource{}
	for _, in := range job.Inputs {
		res, _, err := c.team.Resource(c.pipeline.Name, in.Resource)
		if err != nil {
			c.err = err
			return nil
		}
		result = append(result, res)
	}
	return result
}

func (c CI) PrerequisitesForJob(job atc.Job) map[string][]atc.Job {
	dependencyGraphs, err := findDependencies(c.client, c.pipeline, job.Name)
	if err != nil {
		c.err = err
		return nil
	}
	return dependencyGraphs
}

func (c CI) VersionsForResource(resource atc.Resource) []atc.ResourceVersion {
	versions, _, _, err := c.team.ResourceVersions(c.pipeline.Name, resource.Name, concourse.Page{Limit: 1}, nil)
	if err != nil {
		c.err = err
		return nil
	}
	return versions
}

// LatestBuildsForVersion returns latest build for each job where the specified version has run
func (c CI) LatestBuildsForVersion(resource atc.Resource, version atc.ResourceVersion) []atc.Build {
	builds, _, err := c.team.BuildsWithVersionAsInput(c.pipeline.Name, resource.Name, version.ID)
	if err != nil {
		c.err = err
		return nil
	}
	buildsByJob := make(map[string]atc.Build)
	for _, b := range builds {
		if current, ok := buildsByJob[b.JobName]; ok {
			if current.EndTime < b.EndTime {
				buildsByJob[b.JobName] = b
			}
		} else {
			buildsByJob[b.JobName] = b
		}
	}
	results := make([]atc.Build, len(buildsByJob))
	for _, b := range buildsByJob {
		results = append(results, b)
	}
	return results
}

func getDependentBuildResults(c CI, job atc.Job) []BuildResult {
	output := []BuildResult{}
	promotedResources := c.ResourcesForJob(job)
	allPrereqs := c.PrerequisitesForJob(job)
	for _, res := range promotedResources {
		versions := c.VersionsForResource(res)
		prereqs := allPrereqs[res.Name]
		for _, version := range versions {
			builds := c.LatestBuildsForVersion(res, version)
			output = append(output, matchBuildsToPrereqs(res, version, prereqs, builds)...)
		}
	}
	return output
}

func matchBuildsToPrereqs(resource atc.Resource, version atc.ResourceVersion, prereqs []atc.Job, builds []atc.Build) []BuildResult {
	results := []BuildResult{}

outer:
	for _, job := range prereqs {
		for _, build := range builds {
			if build.JobName == job.Name {
				results = append(results, BuildResult{
					Resource: resource,
					Build:    build,
					Version:  version.Version,
				})
				continue outer
			}
		}
		results = append(results, BuildResult{
			Resource: resource,
			Build: atc.Build{
				JobName: job.Name,
				Status:  "not running",
			},
			Version: version.Version,
		})
	}
	return results
}
