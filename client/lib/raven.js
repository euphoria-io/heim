Raven = require('raven-js')
Raven.config(process.env.SENTRY_ENDPOINT, {
  release: process.env.HEIM_RELEASE,
  tags: {'git_commit': process.env.HEIM_GIT_COMMIT},
}).install()
