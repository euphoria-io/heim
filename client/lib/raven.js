Raven = require('./vendor/raven')
Raven.config(process.env.RAVEN_ENDPOINT).install()
