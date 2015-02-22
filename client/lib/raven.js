Raven = require('./vendor/raven')
Raven.config(process.env.SENTRY_ENDPOINT).install()
