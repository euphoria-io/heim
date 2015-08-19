function writeEnv(doc, hash) {
  var prefix = process.env.HEIM_PREFIX
  var query = hash ? '?v=' + hash : ''
  doc.write('<script src="' + prefix + '/static/raven.js' + query +  '"></script>')
  doc.write('<script src="' + prefix + '/static/main.js' + query +  '"></script>')
  doc.write('<link rel="stylesheet" type="text/css" id="css" href="' + prefix + '/static/main.css' + query + '">')
  doc.write('<link rel="stylesheet" type="text/css" id="emoji-css" href="' + prefix + '/static/emoji.css' + query + '">')
  doc.close()
}

var crashHandlerSetup = false
function setupCrashHandler(evs) {
  if (crashHandlerSetup) {
    return
  }
  var crashHandler = require('./ui/crash-handler')
  evs.addEventListener(document, 'ravenHandle', crashHandler)
  evs.addEventListener(uidocument, 'ravenHandle', crashHandler)
  crashHandlerSetup = true
}

if (!window.frameElement) {
  writeEnv(document.getElementById('env').contentWindow.document)
} else {
  // >:)
  window.uiwindow = window.top
  window.uidocument = window.top.document

  var EventListeners = require('./event-listeners')
  var evs = new EventListeners()

  if (!window.onReady) {
    // if this is the first frame, register crash handlers early
    setupCrashHandler(evs)
  }

  var moment = require('moment')
  moment.relativeTimeThreshold('s', 0)
  moment.relativeTimeThreshold('m', 60)

  moment.locale('en-short', {
    relativeTime: {
      future: 'in %s',
      past: '%s ago',
      s: '%ds',
      m: '1m',
      mm: '%dm',
      h: '1h',
      hh: '%dh',
      d: '1d',
      dd: '%dd',
      M: '1mo',
      MM: '%dmo',
      y: '1y',
      yy: '%dy',
    }
  })

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

  var queryString = require('querystring')
  var _ = require('lodash')
  require('setimmediate')

  var isTextInput = require('./is-text-input')
  var BatchTransition = require('./batch-transition')

  Heim = {
    addEventListener: evs.addEventListener.bind(evs),
    removeEventListener: evs.removeEventListener.bind(evs),

    tabPressed: false,

    setFavicon: function(favicon) { Heim._favicon = favicon },
    setTitleMsg: function(msg) { Heim._titleMsg = msg },

    transition: new BatchTransition(),

    // http://stackoverflow.com/a/6447935
    isTouch: 'ontouchstart' in window,
    isChrome: /chrome/i.test(navigator.userAgent),
    isAndroid: /android/i.test(navigator.userAgent),
    isiOS: /ipad|iphone|ipod/i.test(navigator.userAgent),
  }

  _.extend(Heim, {
    actions: require('./actions'),
    socket: require('./stores/socket'),
    chat: require('./stores/chat'),
    ui: require('./stores/ui'),
    notification: require('./stores/notification'),
    storage: require('./stores/storage'),
    activity: require('./stores/activity'),
    click: require('./stores/clock'),
    update: require('./stores/update'),
    plugins: require('./stores/plugins'),
  })

  Heim.hook = Heim.plugins.hook

  var hashFlags = queryString.parse(location.hash.substr(1))

  if (_.has(hashFlags, 'perf')) {
    var React = require('react/addons')
    if (React.addons && React.addons.Perf) {
      uiwindow.ReactPerf = React.addons.Perf
      uiwindow.ReactPerf.start()
    }
  }

  if (_.has(hashFlags, 'socket')) {
    Heim.socket.store._logPackets = true
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
    setupCrashHandler(evs)

    var Reflux = require('reflux')

    // IE9+ requires this bind: https://msdn.microsoft.com/en-us/library/ie/gg622930(v=vs.85).aspx
    Reflux.nextTick(setImmediate.bind(window))

    var React = require('react/addons')
    var SyntheticKeyboardEvent = require('react/lib/SyntheticKeyboardEvent')
    var Main = require('./ui/main')

    Heim.loadCSS('css')
    Heim.loadCSS('emoji-css')

    Heim.addEventListener(uiwindow, 'storage', Heim.storage.storageChange, false)

    Heim.addEventListener(uiwindow, 'focus', function() {
      Heim.activity.windowFocused()
      Heim.activity.touch(roomName)
    }, false)
    Heim.addEventListener(uiwindow, 'blur', Heim.activity.windowBlurred, false)
    if (uidocument.hasFocus()) {
      Heim.activity.windowFocused()
    }

    Heim.addEventListener(uidocument, 'copy', function(ev) {
      var findParent = require('./find-parent')
      var domWalkForward = require('./dom-walk-forward')
      var emoji = require('./emoji')

      var selection = uiwindow.getSelection()
      var range = selection.getRangeAt(0)

      // first, if the selection start and end are within the same message
      // line, do nothing.
      function findParentMessage(el) {
        return findParent(el, el => el.classList && el.classList.contains('message-node'))
      }
      var startMessageEl = findParentMessage(range.startContainer)
      var endMessageEl = findParentMessage(range.endContainer)
      if (startMessageEl && startMessageEl.querySelector('.line').contains(range.endContainer)) {
        return
      }

      // if the selection start and end aren't within the same message list, do
      // nothing.
      var messagesEl = findParent(startMessageEl, el => el.classList && el.classList.contains('messages'))
      if (!messagesEl || !messagesEl.contains(range.endContainer)) {
        return
      }

      var messageEls = []
      var minDepth
      domWalkForward(startMessageEl, endMessageEl, function (el) {
        if (!el.classList || !el.classList.contains('message-node')) {
          return
        }
        messageEls.push(el)
        if (!minDepth || el.dataset.depth < minDepth) {
          minDepth = el.dataset.depth
        }
      })

      var textParts = []
      _.each(messageEls, el => {
        var messageId = el.dataset.messageId
        var message = Heim.chat.store.state.messages.get(messageId)
        var preContent = []
        preContent.push(_.repeat(' ', 2 * (el.dataset.depth - minDepth)))
        preContent.push('[')
        preContent.push(message.getIn(['sender', 'name']))
        preContent.push('] ')
        preContent = preContent.join('')
        textParts.push(preContent)
        textParts.push(message.get('content').replace(/\n/g, '\n' + _.repeat(' ', preContent.length)))
        textParts.push('\n')
      })

      var text = textParts.join('')
      text = text.replace(emoji.namesRe, (match, name) => emoji.nameToUnicode(name) || match)

      ev.clipboardData.setData('text/plain', text)
      ev.preventDefault()
    }, false)

    Heim.addEventListener(uiwindow, 'message', function(ev) {
      if (ev.origin == process.env.EMBED_ORIGIN) {
        Heim.actions.embedMessage(ev.data)
      }
    }, false)

    Heim.addEventListener(uidocument.body, 'keypress', function(ev) {
      if (!uiwindow.getSelection().isCollapsed) {
        return
      }

      if (isTextInput(ev.target)) {
        return
      }

      if (!ev.which) {
        return
      }

      var character = String.fromCharCode(ev.which)
      if (character) {
        // in Chrome, if we focus synchronously, the input receives the
        // keypress event -- not so in Firefox. we'll delay the focus event to
        // avoid double key insertion in Chrome.
        setImmediate(function() {
          Heim.ui.focusEntry(character)
        })
      }
    }, true)

    Heim.addEventListener(uidocument.body, 'keydown', function(originalEv) {
      Heim.activity.touch(roomName)

      // dig into React a little so it normalizes the event (namely ev.key).
      var ev = new SyntheticKeyboardEvent(null, null, originalEv)

      // prevent alt-left/alt-right back/forward navigation
      if (ev.altKey && (ev.key == 'ArrowLeft' || ev.key == 'ArrowRight')) {
        ev.preventDefault()
      }

      // prevent backspace from navigating the page
      if (ev.key == 'Backspace' && ev.target == uidocument.body) {
        ev.preventDefault()
      }

      if (ev.key == 'Tab') {
        Heim.tabPressed = true
      }

      if (!Heim.mainComponent.getDOMNode().contains(ev.target)) {
        Heim.mainComponent.onKeyDown(ev)
      }
    }, false)

    Heim.addEventListener(uidocument.body, 'keyup', function(originalEv) {
      var ev = new SyntheticKeyboardEvent(null, null, originalEv)
      if (ev.key == 'Tab') {
        Heim.tabPressed = false
      }
    })

    // helpers for catching those pesky mouse-escaped-window-and-released cases
    Heim.addEventListener(uiwindow, 'mouseup', function(ev) {
      Heim.ui.globalMouseUp(ev)
    }, false)

    Heim.addEventListener(uiwindow, 'mousemove', function(ev) {
      Heim.ui.globalMouseMove(ev)
    }, false)

    if (Heim.isTouch) {
      React.initializeTouchEvents()
      uidocument.body.classList.add('touch')

      Heim.addEventListener(uidocument.body, 'touchstart', function(ev) {
        Heim.activity.touch(roomName)
        ev.target.classList.add('touching')
      }, false)

      Heim.addEventListener(uidocument.body, 'touchend', function(ev) {
        ev.target.classList.remove('touching')
      }, false)
    } else {
      Heim.addEventListener(uidocument.body, 'mousedown', function() {
        Heim.activity.touch(roomName)
      }, false)
    }

    Heim.setFavicon = _.partial(require('./set-favicon'), uidocument)
    if (Heim._favicon) {
      Heim.setFavicon(Heim._favicon)
      delete Heim._favicon
    }

    Heim.setTitleMsg = function(msg) {
      uidocument.title = msg ? roomName + ' (' + msg + ')' : roomName
    }
    if (Heim._titleMsg) {
      Heim.setTitleMsg(Heim._titleMsg)
      delete Heim._titleMsg
    }

    setImmediate(function() {
      React.addons.batchedUpdates(() => {
        Heim.mainComponent = React.render(
          <Main />,
          uidocument.getElementById('container')
        )
        uidocument.body.classList.add('ready')
      })
    })
    window.top.Heim = Heim
    window.top.require = require

    Heim.activity.touch(roomName)
  }

  Heim.detachUI = function() {
    uidocument.body.classList.remove('ready')
    evs.removeAllEventListeners()
    Heim.mainComponent.unmountComponent()
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
      context.Heim.actions.connect(roomName, hashFlags.connect)
    }
    writeEnv(context.document, hash)
  }

  Heim.plugins.load(roomName)

  if (!window.onReady) {
    Heim.actions.connect(roomName, hashFlags.connect)
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
