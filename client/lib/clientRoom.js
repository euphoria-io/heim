function writeEnv(doc, hash) {
  const prefix = process.env.HEIM_PREFIX
  const query = hash ? '?v=' + hash : ''
  doc.write('<link rel="stylesheet" type="text/css" id="css" href="' + prefix + '/static/main.css' + query + '">')
  doc.write('<link rel="stylesheet" type="text/css" id="emoji-css" href="' + prefix + '/static/emoji.css' + query + '">')
  doc.write('<script src="' + prefix + '/static/raven.js' + query + '"></script>')
  doc.write('<script id="heim-js" src="' + prefix + '/static/main.js' + query + '"></script>')
  doc.close()
}

let crashHandlerSetup = false
function setupCrashHandler(evs) {
  if (crashHandlerSetup) {
    return
  }
  const crashHandler = require('./ui/crashHandler').default
  evs.addEventListener(document, 'ravenHandle', crashHandler)
  evs.addEventListener(uidocument, 'ravenHandle', crashHandler)
  crashHandlerSetup = true
}

export default function clientRoom() {
  if (!window.frameElement) {
    writeEnv(document.getElementById('env').contentWindow.document, process.env.HEIM_GIT_COMMIT)
  } else {
    const queryString = require('querystring')
    const _ = require('lodash')
    const EventListeners = require('./EventListeners').default

    const evs = new EventListeners()
    if (!window.onReady) {
      // if this is the first frame, register crash handlers early
      setupCrashHandler(evs)
    }

    // read url hash flags pertaining to socket connection
    const roomName = uiwindow.location.pathname.match(/((pm:)?\w+)\/$/)[1]
    const hashFlags = queryString.parse(uiwindow.location.hash.substr(1))
    let connectEndpoint = process.env.HEIM_ORIGIN + process.env.HEIM_PREFIX
    if (process.env.NODE_ENV !== 'production' && hashFlags.connect) {
      connectEndpoint = hashFlags.connect
    }
    const socketLog = _.has(hashFlags, 'socket')

    // connect websocket as early as possible so we can start streaming data
    const Socket = require('./heim/Socket').default
    const socket = new Socket()
    socket.startBuffering()
    socket.connect(connectEndpoint, roomName, {log: socketLog})

    // set up general environment
    const moment = require('moment')
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
      },
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
      },
    })

    const isTextInput = require('./isTextInput').default
    const BatchTransition = require('./BatchTransition').default

    window.Heim = {
      addEventListener: evs.addEventListener.bind(evs),
      removeEventListener: evs.removeEventListener.bind(evs),

      tabPressed: false,

      setFavicon(favicon) { Heim._favicon = favicon },
      setTitleMsg(msg) { Heim._titleMsg = msg },
      setTitlePrefix(prefix) { Heim._titlePrefix = prefix },
      _getTitlePrefix() { return Heim._titlePrefix },

      transition: new BatchTransition(),

      // http://stackoverflow.com/a/6447935
      isTouch: 'ontouchstart' in window,
      isChrome: /chrome/i.test(navigator.userAgent),
      isAndroid: /android/i.test(navigator.userAgent),
      isiOS: /ipad|iphone|ipod/i.test(navigator.userAgent),

      socket: {
        devSend(packet) {
          Heim.chat.store.socket.send(packet, true)
        },
      },
    }

    _.extend(Heim, {
      actions: require('./actions'),
      chat: require('./stores/chat'),
      ui: require('./stores/ui'),
      notification: require('./stores/notification'),
      storage: require('./stores/storage'),
      activity: require('./stores/activity'),
      click: require('./stores/clock'),
      update: require('./stores/update'),
      plugins: require('./stores/plugins'),
    })

    Heim.chat.store.socket = socket

    Heim.hook = Heim.plugins.hook

    if (_.has(hashFlags, 'perf')) {
      uiwindow.ReactPerf = require('react-addons-perf')
      uiwindow.ReactPerf.start()
    }

    Heim.loadCSS = function loadCSS(id) {
      const cssEl = uidocument.getElementById(id)
      const cssURL = document.getElementById(id).getAttribute('href')
      if (!cssEl || cssEl.parentNode !== uidocument.head || cssEl.getAttribute('href') !== cssURL) {
        const newCSSEl = uidocument.createElement('link')
        newCSSEl.id = id
        newCSSEl.rel = 'stylesheet'
        newCSSEl.type = 'text/css'
        newCSSEl.href = cssURL
        uidocument.head.appendChild(newCSSEl)

        if (cssEl) {
          cssEl.id = id + '-old'

          // allow both stylesheets to coexist briefly in an attempt to avoid FOUSC
          setTimeout(() => cssEl.parentNode.removeChild(cssEl), 30)
        }
      }
    }

    Heim.attachUI = function attachUI() {
      setupCrashHandler(evs)

      const Reflux = require('reflux')

      // IE9+ requires this bind: https://msdn.microsoft.com/en-us/library/ie/gg622930(v=vs.85).aspx
      Reflux.nextTick(setImmediate.bind(window))

      const React = require('react')
      const ReactDOM = require('react-dom')
      const SyntheticKeyboardEvent = require('react/lib/SyntheticKeyboardEvent')
      const Main = require('./ui/Main').default

      Heim.loadCSS('css')
      Heim.loadCSS('emoji-css')

      Heim.addEventListener(uiwindow, 'storage', Heim.storage.storageChange, false)

      Heim.addEventListener(uiwindow, 'focus', () => {
        Heim.activity.windowFocused()
        Heim.activity.touch(roomName)
      }, false)
      Heim.addEventListener(uiwindow, 'blur', () => {
        Heim.activity.windowBlurred()
        Heim.tabPressed = false
      }, false)
      if (uidocument.hasFocus()) {
        Heim.activity.windowFocused()
      }

      Heim.addEventListener(uiwindow, 'message', ev => {
        if (ev.origin === process.env.EMBED_ORIGIN) {
          Heim.actions.embedMessage(ev.data)
        }
      }, false)

      Heim.addEventListener(uidocument.body, 'keypress', ev => {
        if (!uiwindow.getSelection().isCollapsed) {
          return
        }

        if (isTextInput(ev.target)) {
          return
        }

        if (!ev.which) {
          return
        }

        const character = String.fromCharCode(ev.which)
        if (character) {
          // in Chrome, if we focus synchronously, the input receives the
          // keypress event -- not so in Firefox. we'll delay the focus event to
          // avoid double key insertion in Chrome.
          setImmediate(() => Heim.ui.focusEntry(character))
        }
      }, true)

      Heim.addEventListener(uidocument.body, 'keydown', originalEv => {
        Heim.activity.touch(roomName)

        // dig into React a little so it normalizes the event (namely ev.key).
        const ev = new SyntheticKeyboardEvent(null, null, originalEv, originalEv.target)

        // prevent backspace from navigating the page
        if (ev.key === 'Backspace' && ev.target === uidocument.body) {
          ev.preventDefault()
        }

        if (ev.key === 'Tab') {
          Heim.tabPressed = true
        }

        if (Heim.mainComponent && !ReactDOM.findDOMNode(Heim.mainComponent).contains(ev.target)) {
          Heim.ui.keydownOnPage(ev)
        }
      }, false)

      Heim.addEventListener(uidocument.body, 'keyup', originalEv => {
        const ev = new SyntheticKeyboardEvent(null, null, originalEv)
        if (ev.key === 'Tab') {
          Heim.tabPressed = false
        }
      })

      // helpers for catching those pesky mouse-escaped-window-and-released cases
      Heim.addEventListener(uiwindow, 'mouseup', ev => Heim.ui.globalMouseUp(ev), false)
      Heim.addEventListener(uiwindow, 'mousemove', ev => Heim.ui.globalMouseMove(ev), false)

      if (Heim.isTouch) {
        uidocument.body.classList.add('touch')

        Heim.addEventListener(uidocument.body, 'touchstart', ev => {
          Heim.activity.touch(roomName)
          ev.target.classList.add('touching')
        }, false)

        Heim.addEventListener(uidocument.body, 'touchend', ev => {
          ev.target.classList.remove('touching')
        }, false)
      } else {
        Heim.addEventListener(uidocument.body, 'mousedown', () => Heim.activity.touch(roomName), false)
      }

      Heim.setFavicon = _.partial(require('./setFavicon').default, uidocument)
      if (Heim._favicon) {
        Heim.setFavicon(Heim._favicon)
        delete Heim._favicon
      }

      Heim._getTitlePrefix = () => { return Heim._titlePrefix || roomName }
      Heim.setTitleMsg = msg => uidocument.title = msg ? Heim._getTitlePrefix() + ' (' + msg + ')' : Heim._getTitlePrefix()
      if (Heim._titleMsg) {
        Heim.setTitleMsg(Heim._titleMsg)
        delete Heim._titleMsg
      }

      Heim.mainComponent = ReactDOM.render(
        <Main />,
        uidocument.getElementById('container')
      )
      uidocument.body.classList.add('ready')
      _.identity(uidocument.body.clientHeight)
      uidocument.body.classList.add('visible')

      window.top.Heim = Heim
      window.top.require = require

      Heim.activity.touch(roomName)
    }

    Heim.detachUI = function detachUI() {
      const ReactDOM = require('react-dom')
      uidocument.body.classList.remove('ready', 'visible')
      evs.removeAllEventListeners()
      ReactDOM.unmountComponentAtNode(uidocument.getElementById('container'))
    }

    Heim.prepareUpdate = function prepareUpdate(hash) {
      Heim.update.setReady(false)

      const oldFrame = uidocument.getElementById('env-update')
      if (oldFrame) {
        oldFrame.parentNode.removeChild(oldFrame)
      }

      const frame = uidocument.createElement('iframe')
      frame.id = 'env-update'
      frame.className = 'js'
      uidocument.body.appendChild(frame)

      frame.contentDocument.open()
      const context = frame.contentWindow
      context.onReady = function onReady() {
        const removeListener = context.Heim.chat.store.listen(function preUpdateChatHandler(chatState) {
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
        context.Heim.actions.connect()
      }
      writeEnv(context.document, hash)
    }

    Heim.plugins.load(roomName)
    Heim.actions.setup(roomName)

    setImmediate(() => {
      if (window.onReady) {
        window.onReady()
      } else {
        Heim.attachUI()
        Heim.actions.joinRoom()
        Heim.actions.connect()
      }
    })
  }
}
