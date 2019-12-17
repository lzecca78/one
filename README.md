# one

Multiverse backend for multistaging environments

## Authors

This project was developed by me and https://github.com/phisco

## Purpose

The problem that this service solves is: we want to create multiple staging environment in order to test in end2end scenario, different branch of each microservice (repository).

The initial trigger for this service was the need to have a service that:

1. given a list of repositories
2. the user is able to choose for each repository, a branch
3. the service has to create a staging environment with all accessories services (db, ingresses, services) and the selected (repository, branch).

## Constraints/Limitations

The project is not able (at the time of writing) to handle multiple providers, and general purpose architectural scenarios.
Due to the company context related scenario in which was developed it has the following constraints:

1. the cloud provider is AWS
2. the ci/cd service is Jenkins
3. the cvs is github

i know that are _big_ constraints, but at least is a good starting point, in our company it helped a lot.

## Future development

We would like to carry forward the project trying to implement interfaces for each current static implementation in order to make it agnostic as much as possible.

## Issues and pull requests

Please, feel free to open pull requests, issues to help us to make this project better.
