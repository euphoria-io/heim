function writeEnv(doc, hash) {
  var prefix = process.env.HEIM_PREFIX
  var query = hash ? '?v=' + hash : ''
  doc.write('<script src="' + prefix + '/static/raven.js' + query +  '"></script>')
  doc.write('<script src="' + prefix + '/static/main.js' + query +  '"></script>')
  doc.write('<link rel="stylesheet" type="text/css" id="css" href="' + prefix + '/static/main.css' + query + '">')
  doc.write('<link rel="stylesheet" type="text/css" id="emoji-css" href="' + prefix + '/static/emoji.css' + query + '">')
  doc.close()
}

if (!window.frameElement) {
  writeEnv(document.getElementById('env').contentWindow.document)
} else {
  // >:)
  window.uiwindow = window.top
  window.uidocument = window.top.document


  var moment = require('moment')
  moment.locale('en', {
    relativeTime: {
      future: 'in %s',
      past: '%s ago',
      s: '%ds',
      m: '1 min',
      mm: '%d min',
      h: '1 hour',
      hh: '%d hours',
      d: 'a day',
      dd: '%d days',
      M: 'a month',
      MM: '%d months',
      y: 'a year',
      yy: '%d years',
    }
  })
  moment.relativeTimeThreshold('s', 0)
  moment.relativeTimeThreshold('m', 60)


  var _ = require('lodash')
  require('setimmediate')

  var EventListeners = require('./event-listeners')
  var isTextInput = require('./is-text-input')

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

    setFavicon: _.partial(require('./set-favicon'), uidocument),

    // http://stackoverflow.com/a/6447935
    isTouch: 'ontouchstart' in window,
    isChrome: /chrome/i.test(navigator.userAgent),
    isAndroid: /android/i.test(navigator.userAgent),
    isiOS: /ipad|iphone|ipod/i.test(navigator.userAgent),
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

  Heim.loadCSS = function(id) {
    var cssEl = uidocument.getElementById(id)
    var cssURL = document.getElementById(id).getAttribute('href')
    if (!cssEl || cssEl.parentNode != uidocument.head || cssEl.getAttribute('href') != cssURL) {
      var newCSSEl = uidocument.createElement('link')
      newCSSEl.id = id
      newCSSEl.rel = 'stylesheet'
      newCSSEl.type = 'text/css'
      newCSSEl.href = cssURL
      uidocument.head.appendChild(newCSSEl)

      if (cssEl) {
        cssEl.id = id + '-old'

        // allow both stylesheets to coexist briefly in an attempt to avoid FOUSC
        setTimeout(function() {
          cssEl.parentNode.removeChild(cssEl)
        }, 30)
      }
    }
  }

  Heim.attachUI = function() {
    var Reflux = require('reflux')

    // IE9+ requires this bind: https://msdn.microsoft.com/en-us/library/ie/gg622930(v=vs.85).aspx
    Reflux.nextTick(setImmediate.bind(window))

    var React = require('react/addons')
    var SyntheticKeyboardEvent = require('react/lib/SyntheticKeyboardEvent')
    var Main = require('./ui/main')

    uidocument.title = roomName

    Heim.loadCSS('css')
    Heim.loadCSS('emoji-css')

    Heim.addEventListener(uiwindow, 'storage', Heim.storage.storageChange, false)

    Heim.addEventListener(uiwindow, 'focus', Heim.focus.windowFocused, false)
    Heim.addEventListener(uiwindow, 'blur', Heim.focus.windowBlurred, false)
    if (uidocument.hasFocus()) {
      Heim.focus.windowFocused()
    }

    Heim.addEventListener(uiwindow, 'message', function(ev) {
      if (ev.origin == process.env.EMBED_ORIGIN) {
        Heim.actions.embedMessage(ev.data)
      }
    }, false)

    Heim.addEventListener(uidocument.body, 'keypress', function(ev) {
      if (!uiwindow.getSelection().isCollapsed){
        return
      }

      if (isTextInput(ev.target)) {
        return
      }

      if (!ev.which) {
        return
      }

      var character = String.fromCharCode(ev.which)
      if (character && /\S/.test(character)) {
        // in Chrome, if we focus synchronously, the input receives the
        // keypress event -- not so in Firefox. we'll delay the focus event to
        // avoid double key insertion in Chrome.
        setImmediate(function() {
          Heim.actions.focusEntry(character)
        })
      }
    }, true)

    Heim.addEventListener(uidocument.body, 'keydown', function(ev) {
      if (isTextInput(ev.target)) {
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
      uidocument.body.classList.add('ready')
    })
    window.top.Heim = Heim
    window.top.require = require
  }

  Heim.detachUI = function() {
    uidocument.body.classList.remove('ready')
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
    writeEnv(context.document, hash)
  }

  Heim.plugins.load(roomName)

  if (!window.onReady) {
    Heim.actions.connect(roomName)
    Heim.actions.joinRoom()
  }

  setImmediate(function() {
    if (window.onReady) {
      window.onReady()
    } else {
      Heim.attachUI()
    }
  })
}
