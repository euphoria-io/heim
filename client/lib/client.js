var React = require('react/addons')
var SyntheticKeyboardEvent = require('react/lib/SyntheticKeyboardEvent')

var Main = require('./ui/main')


Heim = {
  actions: require('./actions'),
  socket: require('./stores/socket'),
  chat: require('./stores/chat'),
  notification: require('./stores/notification'),
  storage: require('./stores/storage'),
  focus: require('./stores/focus'),
  // http://stackoverflow.com/a/6447935
  isTouch: 'ontouchstart' in window,
}

var roomName = location.pathname.match(/(\w+)\/$/)[1]
document.title = roomName
Heim.actions.connect(roomName)

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

document.body.addEventListener('keypress', function(ev) {
  if (ev.target.nodeName == 'INPUT' && ev.target.type == 'text') {
    return
  }

  if (!ev.which) {
    return
  }

  var character = String.fromCharCode(ev.which)
  if (character && /\S/.test(character)) {
    Heim.actions.focusEntry(character)
  }
}, true)

document.body.addEventListener('keydown', function(ev) {
  if (ev.target.nodeName == 'INPUT') {
    return
  }

  // prevent backspace from navigating the page
  if (ev.which == 8) {
    ev.preventDefault()
  }

  // dig into React a little so it normalizes the event (namely ev.key).
  var reactEvent = new SyntheticKeyboardEvent(null, null, ev)
  Heim.actions.keydownOnEntry(reactEvent)
}, false)

if (Heim.isTouch) {
  React.initializeTouchEvents()
  document.body.classList.add('touch')

  document.body.addEventListener('touchstart', function(ev) {
    ev.target.classList.add('touching')
  }, false)

  document.body.addEventListener('touchend', function(ev) {
    ev.target.classList.remove('touching')
  }, false)
}
