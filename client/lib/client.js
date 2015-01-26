var React = require('react/addons')

var Main = require('./ui/main')

var roomName = location.pathname.match(/(\w+)\/$/)[1]
document.title = roomName
require('./actions').connect(roomName)

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

Heim = {
  actions: require('./actions'),
  socket: require('./stores/socket'),
  chat: require('./stores/chat'),
  notification: require('./stores/notification'),
  storage: require('./stores/storage'),
  focus: require('./stores/focus'),
}

document.body.addEventListener('keypress', function(ev) {
  if (ev.target.nodeName == 'INPUT' && ev.target.type == 'text') {
    return
  }

  var character = String.fromCharCode(ev.which)
  if (character) {
    Heim.actions.focusEntry(character)
  }
}, true)

// prevent backspace from navigating the page
document.body.addEventListener('keydown', function(ev) {
  if (ev.target.nodeName != 'INPUT' && ev.which == 8) {
    ev.preventDefault()
  }
}, false)
