# Concourse Blockers

This tool will query the concourse API and tell show you which jobs must pass for a given job to trigger.

I use it to debug complicated passed / trigger constraints in big pipelines.

## Usage

It will make use of your saved targets in the `fly` cli. Assuming you have authenticated with `fly login`, this tool should work.

```
$ concourse-blockers
Usage of concourse-blockers:
  -j string
    	OPTIONAL, Concourse job, e.g. build
  -p string
    	Concourse pipeline, e.g. master
  -r string
    	OPTIONAL, Concourse resource, e.g. repo-name
  -t string
    	Concourse target, e.g. ci
```

For example:

```
$ concourse-blockers -t dev -p master -j deploy_pas_tiles -r om-yml
deploy_pas_tiles
  om-yml
    The following jobs must pass for this trigger to occur:
    import_om_ami, tf_apply_pas_ops_manager, create-pas-db, configure-pas-bosh-director, deploy-pas-bosh-director, configure_pas_and_clamav, deploy_pas_and_clamav, configure_pas_tiles, deploy_pas_tiles
```

The output is color coded to show which jobs in the path will / won't trigger.
