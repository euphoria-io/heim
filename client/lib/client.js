var React = require('react')
require('superagent-bluebird-promise')

var Main = require('./ui/main')


React.render(
  <Main />,
  document.getElementById('container')
)

require('./actions').connect()

document.t abody.classList.add('loaded')
