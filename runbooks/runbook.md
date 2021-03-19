<!--
    Written in the format prescribed by https://github.com/Financial-Times/runbook.md.
    Any future edits should abide by this format.
-->
# UPP - Mongo Hot Backup

This tool can back up or restore MongoDB collections while DB is running to/from AWS S3. You can deploy a docker container that will run backups on schedule. Or you can just run the container to make a single backup, or restore from a given point of time.

## Code

mongo-hot-backup

## Primary URL

https://github.com/Financial-Times/mongo-hot-backup

## Service Tier

Bronze

## Lifecycle Stage

Production

## Host Platform

AWS

## Architecture

For the schedule option, the state of backups is kept in a boltdb file. (at /var/data/mongo-hot-backup/state.db or where you set it)

Health endpoint is available at 0.0.0.0:8080/**health and will report healthy if there was a successful backup for each configured collection in the last X hours, also configurable. Good-to-go /**gtg endpoint available as well, and /build-info.

An initial backup to be ran upon startup can be enabled.

## Contains Personal Data

No

## Contains Sensitive Data

No

<!-- Placeholder - remove HTML comment markers to activate
## Can Download Personal Data
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

<!-- Placeholder - remove HTML comment markers to activate
## Can Contact Individuals
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

## Failover Architecture Type

ActiveActive

## Failover Process Type

NotApplicable

## Failback Process Type

NotApplicable

## Failover Details

NotApplicable

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

NotApplicable

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

The release is triggered by making a Github release which is then picked up by a Jenkins multibranch pipeline. The Jenkins pipeline should be manually started in order for it to deploy the helm package to the Kubernetes clusters.

<!-- Placeholder - remove HTML comment markers to activate
## Heroku Pipeline Name
Enter descriptive text satisfying the following:
This is the name of the Heroku pipeline for this system. If you don't have a pipeline, this is the name of the app in Heroku. A pipeline is a group of Heroku apps that share the same codebase where each app in a pipeline represents the different stages in a continuous delivery workflow, i.e. staging, production.

...or delete this placeholder if not applicable to this system
-->

## Key Management Process Type

NotApplicable

## Key Management Details

There is no key rotation procedure for this system.

## Monitoring

Pod health:

*   <https://upp-prod-publish-eu.upp.ft.com/__health/__pods-health?service-name=mongo-hot-backup>
*   <https://upp-prod-publish-us.upp.ft.com/__health/__pods-health?service-name=mongo-hot-backup>
*   <https://upp-prod-delivery-eu.upp.ft.com/__health/__pods-health?service-name=mongo-hot-backup>
*   <https://upp-prod-delivery-us.upp.ft.com/__health/__pods-health?service-name=mongo-hot-backup>

## First Line Troubleshooting

[First Line Troubleshooting guide](https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting)

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.