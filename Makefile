test:
	source ../.envrc && docker run --env ONE_GITHUB_TOKEN=${ONE_GITHUB_TOKEN} one:test
