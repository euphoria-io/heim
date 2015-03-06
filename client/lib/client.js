// >:)
window.uiwindow = window.top
window.uidocument = window.top.document

var _ = require('lodash')
require('setimmediate')

var EventListeners = require('./eventlisteners')


var evs = new EventListeners()

Heim = {
  addEventListener: evs.addEventListener.bind(evs),
  removeEventListener: evs.removeEventListener.bind(evs),

  actions: require('./actions'),
  socket: require('./stores/socket'),
  chat: require('./stores/chat'),
  notification: require('./stores/notification'),
  storage: require('./stores/storage'),
  focus: require('./stores/focus'),
  update: require('./stores/update'),
  plugins: require('./stores/plugins'),

  setFavicon: _.partial(require('./setfavicon'), uidocument),

  // http://stackoverflow.com/a/6447935
  isTouch: 'ontouchstart' in window,
  isChrome: /chrome/i.test(navigator.userAgent),
  isAndroid: /android/i.test(navigator.userAgent),
}

Heim.hook = Heim.plugins.hook

if (location.hash == '#perf') {
  var React = require('react/addons')
  if (React.addons && React.addons.Perf) {
    uiwindow.ReactPerf = React.addons.Perf
    uiwindow.ReactPerf.start()
  }
}

var roomName = location.pathname.match(/(\w+)\/$/)[1]

Heim.attachUI = function(hash) {
  var Reflux = require('reflux')
  Reflux.nextTick(setImmediate)

  var React = require('react/addons')
  var SyntheticKeyboardEvent = require('react/lib/SyntheticKeyboardEvent')
  var Main = require('./ui/main')

  uidocument.title = roomName

  var cssEl = uidocument.getElementById('css')
  var cssURL = '/static/main.css' + (hash ? '?v=' + hash : '')
  if (cssEl.parentNode != uidocument.head || cssEl.getAttribute('href') != cssURL) {
    var newCSSEl = cssEl.cloneNode()
    newCSSEl.href = cssURL
    cssEl.id = 'css-old'
    uidocument.head.appendChild(newCSSEl)

    // allow both stylesheets to coexist briefly in an attempt to avoid FOUSC
    setTimeout(function() {
      cssEl.parentNode.removeChild(cssEl)
    }, 30)
  }

  Heim.addEventListener(uiwindow, 'storage', Heim.storage.storageChange, false)

  Heim.addEventListener(uiwindow, 'focus', Heim.focus.windowFocused, false)
  Heim.addEventListener(uiwindow, 'blur', Heim.focus.windowBlurred, false)
  if (uidocument.hasFocus()) {
    Heim.focus.windowFocused()
  }

  Heim.addEventListener(uidocument.body, 'keypress', function(ev) {
    if (ev.target.nodeName == 'INPUT' &&
         (ev.target.type == 'text' || ev.target.type == 'password')) {
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

  Heim.addEventListener(uidocument.body, 'keydown', function(ev) {
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
    uidocument.body.classList.add('touch')

    Heim.addEventListener(uidocument.body, 'touchstart', function(ev) {
      ev.target.classList.add('touching')
    }, false)

    Heim.addEventListener(uidocument.body, 'touchend', function(ev) {
      ev.target.classList.remove('touching')
    }, false)
  }

  setImmediate(function() {
    Heim.ui = React.render(
      <Main />,
      uidocument.getElementById('container')
    )
  })
  window.top.Heim = Heim
  window.top.require = require
}

Heim.detachUI = function() {
  evs.removeAllEventListeners()
  Heim.ui.unmountComponent()
}

Heim.prepareUpdate = function(hash) {
  Heim.update.setReady(false)

  var frame = uidocument.getElementById('env-update')
  if (frame) {
    frame.parentNode.removeChild(frame)
  }

  frame = uidocument.createElement('iframe')
  frame.id = 'env-update'
  frame.className = 'js'
  uidocument.body.appendChild(frame)

  frame.contentDocument.open()
  var context = frame.contentWindow
  context.onReady = function() {
    var removeListener = context.Heim.chat.store.listen(function(chatState) {
      if (chatState.joined) {
        removeListener()

        // let go of #container
        Heim.detachUI()

        // attach new React component to #container
        context.Heim.attachUI(hash)
        frame.id = 'env'

        // goodbye, world!
        window.frameElement.parentNode.removeChild(window.frameElement)
      } else if (chatState.canJoin) {
        Heim.update.setReady(true, context.Heim.actions.joinRoom)
      } else {
        Heim.update.setReady(false)
      }
    })
    context.Heim.actions.connect(roomName)
  }
  context.document.write('<script src="/static/main.js?v=' + hash +  '"></sc'+'ript>')
  context.document.write('<link rel="stylesheet" type="text/css" id="css" href="/static/main.css?v=' + hash + '">')
  context.document.close()
}

if (!window.onReady) {
  Heim.actions.connect(roomName)
  Heim.actions.joinRoom()
}

setImmediate(function() {
  Heim.plugins.load()

  if (window.onReady) {
    window.onReady()
  } else {
    Heim.attachUI()
  }
})
