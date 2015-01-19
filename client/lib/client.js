var React = require('react/addons')

var Main = require('./ui/main')


if (React.addons && React.addons.Perf) {
  ReactPerf = React.addons.Perf
  if (location.hash == '#perf') {
    ReactPerf.start()
  }
}

React.render(
  <Main />,
  document.getElementById('container')
)

require('./actions').connect()

Heim = {
  actions: require('./actions'),
  socket: require('./stores/socket'),
  chat: require('./stores/chat'),
  storage: require('./stores/storage'),
  focus: require('./stores/focus'),
}

document.title = location.pathname.match(/(\/\w+)\/$/)[1]
