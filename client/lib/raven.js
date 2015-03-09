Raven = require('raven-js')
Raven.config(process.env.SENTRY_ENDPOINT).install()
