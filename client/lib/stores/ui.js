var _ = require('lodash')
var Reflux = require('reflux')
var React = require('react/addons')
var Immutable = require('immutable')

var clamp = require('../clamp')
var actions = require('../actions')
var storage = require('./storage')
var chat = require('./chat')
var notification = require('./notification')
var MessageData = require('../message-data')


var storeActions = module.exports.actions = Reflux.createActions([
  'keydownOnPage',
  'setUIMode',
  'focusEntry',
  'focusPane',
  'focusLeftPane',
  'focusRightPane',
  'collapseInfoPane',
  'expandInfoPane',
  'freezeInfo',
  'thawInfo',
  'selectThread',
  'deselectThread',
  'selectThreadInList',
  'panViewTo',
  'onViewPan',
  'popupToThreadPane',
  'openThreadPane',
  'closeThreadPane',
  'closeFocusedThreadPane',
  'gotoMessageInPane',
  'gotoPopupMessage',
])
_.extend(module.exports, storeActions)

// sync to allow entry to preventDefault keydown events
storeActions.keydownOnPage.sync = true

// sync so UI mode changes don't flicker on load
storeActions.setUIMode.sync = true

// sync so that UI pans start animating immediately
storeActions.panViewTo.sync = true

// temporarily sync while testing this commit
storeActions.focusPane.sync = true

var store = module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    storeActions,
    {storageChange: storage.store},
    {chatChange: chat.store},
    {notificationChange: notification.store},
  ],

  init: function() {
    this.state = {
      thin: false,
      focusedPane: 'main',
      panes: Immutable.Map({
        main: createPaneStore('main'),
      }),
      visiblePanes: Immutable.OrderedSet(['main']),
      popupPane: null,
      infoPaneExpanded: false,
      frozenThreadList: null,
      frozenNotifications: null,
      selectedThread: null,
      lastSelectedThread: null,
      threadPopupAnchorEl: null,
    }

    this.threadData = new MessageData({selected: false})

    this._thawInfoDebounced = _.debounce(this._thawInfo, 1500)
  },

  getInitialState: function() {
    return this.state
  },

  storageChange: function(data) {
    if (!data) {
      return
    }
    this.state.infoPaneExpanded = _.get(data, ['room', this.chatState.roomName, 'infoPaneExpanded'], false)
    this.trigger(this.state)
  },

  chatChange: function(state) {
    this.chatState = state
  },

  notificationChange: function(state) {
    this.notificationState = state
  },

  setUIMode: function(mode) {
    this.state.thin = mode.thin
    this.trigger(this.state)
  },

  collapseInfoPane: function() {
    storage.setRoom(this.chatState.roomName, 'infoPaneExpanded', false)
  },

  expandInfoPane: function() {
    storage.setRoom(this.chatState.roomName, 'infoPaneExpanded', true)
  },

  freezeInfo: function() {
    this._thawInfoDebounced.cancel()
    if (!this.state.frozenThreadList || !this.state.frozenNotifications) {
      this.state.frozenThreadList = this.chatState.messages.threads.clone()
      this.state.frozenNotifications = this.notificationState.notifications
      this.trigger(this.state)
    }
  },

  thawInfo: function() {
    this._thawInfoDebounced()
  },

  _thawInfo: function() {
    if (this.state.threadPopupAnchorEl) {
      return
    }
    this.state.frozenThreadList = null
    this.state.frozenNotifications = null
    this.trigger(this.state)
  },

  _touchThreadPane: function(threadId) {
    var paneId = 'thread-' + threadId
    if (!this.state.panes.has(paneId)) {
      this.state.panes = this.state.panes.set(paneId, createPaneStore(paneId, {rootId: threadId}))
    }
    var pane = this.state.panes.get(paneId)
    pane.focusMessage(threadId)
    return pane
  },

  selectThread: function(id, el) {
    React.addons.batchedUpdates(() => {
      var pane = this._touchThreadPane(id)
      if (this.state.selectedThread && this.state.selectedThread != id) {
        this.threadData.set(this.state.selectedThread, {selected: false})
      }
      this.threadData.set(id, {selected: true})
      this.state.selectedThread = id
      if (this.state.thin) {
        storeActions.panViewTo('main')
        this.focusPane(pane.id, {focusEntry: false})
      } else {
        this.freezeInfo()
        this.state.popupPane = pane.id
        this.state.threadPopupAnchorEl = el
        this.focusPane(pane.id)
      }
      this.trigger(this.state)
    })
  },

  deselectThread: function() {
    if (!this.state.selectedThread) {
      return
    }
    React.addons.batchedUpdates(() => {
      this.threadData.set(this.state.selectedThread, {selected: false})
      this.state.lastSelectedThread = this.state.selectedThread
      this.state.selectedThread = null
      if (this.state.thin) {
        storeActions.focusPane('main', {focusEntry: false})
      } else {
        if (!this.state.popupPane) {
          return
        }
        var popupPane = this.state.panes.get(this.state.popupPane)
        if (popupPane.store.state.entryText.length) {
          return
        }
        this.thawInfo()
        this.state.threadPopupAnchorEl = null
        this.state.popupPane = null
      }
      storeActions.panViewTo('main')
      this.trigger(this.state)
    })
  },

  openThreadPane: function(threadId) {
    React.addons.batchedUpdates(() => {
      var pane = this._touchThreadPane(threadId)
      this.state.visiblePanes = this.state.visiblePanes.add(pane.id)
      this.deselectThread()
      this.chatState.messages.mergeNodes(threadId, {_inPane: pane.id})
      this.focusPane(pane.id)
      this.trigger(this.state)
    })
  },

  popupToThreadPane: function() {
    this.openThreadPane(this.state.selectedThread)
  },

  closeThreadPane: function(threadId) {
    React.addons.batchedUpdates(() => {
      var paneId = 'thread-' + threadId
      this.focusPane('main')
      this.state.panes = this.state.panes.delete(paneId)
      this.state.visiblePanes = this.state.visiblePanes.remove(paneId)
      this.chatState.messages.mergeNodes(threadId, {_inPane: false})
      this.trigger(this.state)
    })
  },

  closeFocusedThreadPane: function() {
    if (!/^thread-/.test(this.state.focusedPane)) {
      return
    }
    this.closeThreadPane(this._focusedPane().store.state.rootId)
  },

  _focusedPane: function() {
    return this.state.panes.get(this.state.focusedPane)
  },

  focusPane: function(id, opts) {
    if (this.state.focusedPane == id) {
      return
    }

    opts = opts || {}
    React.addons.batchedUpdates(() => {
      var lastFocused = this.state.focusedPane
      this.state.focusedPane = id
      this.trigger(this.state)

      require('react/lib/ReactUpdates').asap(() => {
        var lastFocusedPane = this.state.panes.get(lastFocused)
        if (lastFocusedPane) {
          // the pane has been removed while the batching occurred
          lastFocusedPane.blurEntry()
        }
        if (opts.focusEntry !== false) {
          this.state.panes.get(id).focusEntry()
        }
      })
    })
  },

  _moveFocusedPane: function(delta) {
    var focusablePanes = this.state.visiblePanes
      .toKeyedSeq()
      .map(paneId => this.state.panes.get(paneId))
      .filterNot(pane => pane.readOnly)
      .cacheResult()
    var idx = focusablePanes.keySeq().indexOf(this.state.focusedPane)
    idx = clamp(-1, idx + delta, focusablePanes.size - 1)

    if (idx == -1) {
      storeActions.panViewTo('info')
    } else {
      storeActions.panViewTo('main')
      var paneId = focusablePanes.entrySeq().get(idx)[0]
      this.focusPane(paneId)
    }
  },

  focusLeftPane: function() {
    this._moveFocusedPane(-1)
  },

  focusRightPane: function() {
    this._moveFocusedPane(1)
  },

  focusEntry: function(character) {
    this._focusedPane().focusEntry(character)
  },

  onViewPan: function(target) {
    if (!this.state.thin) {
      if (target == 'info') {
        if (!this.state.selectedThread) {
          storeActions.selectThreadInList(this.state.lastSelectedThread)
          return
        }
      } else {
        this.deselectThread()
      }
    }
  },

  keydownOnPage: function(ev) {
    this._focusedPane().keydownOnPane(ev)
  },

  gotoMessageInPane: function(messageId) {
    var parentPaneId = Immutable.Seq(this.chatState.messages.iterAncestorsOf(messageId))
      .map(ancestor => 'thread-' + ancestor.get('id'))
      .find(threadId => this.state.visiblePanes.has(threadId))

    var parentPane = this.state.panes.get(parentPaneId || 'main')

    React.addons.batchedUpdates(() => {
      parentPane.revealMessage(messageId)
      parentPane.focusMessage(messageId)
    })
  },

  gotoPopupMessage: function() {
    var mainPane = this.state.panes.get('main')
    React.addons.batchedUpdates(() => {
      mainPane.revealMessage(this.state.selectedThread)
      mainPane.focusMessage(this.state.selectedThread)
      this.deselectThread()
    })
  },

  _createCustomPane: function(paneId, options) {
    var newPane = createPaneStore(paneId, options)
    this.state.panes = this.state.panes.set(paneId, newPane)
    this.trigger(this.state)
    return newPane
  },
})

module.exports.createCustomPane = function(paneId, options) {
  return store._createCustomPane(paneId, options)
}

function createPaneStore(paneId, createOptions) {
  createOptions = createOptions || {}

  var paneActions = Reflux.createActions([
    'sendMessage',
    'focusMessage',
    'toggleFocusMessage',
    'moveMessageFocus',
    'revealMessage',
    'escape',
    'focusEntry',
    'blurEntry',
    'scrollToEntry',
    'messageRenderFinished',
    'afterMessagesRendered',
    'setMessageData',
    'setEntryText',
    'keydownOnPane',
    'openFocusedMessageInPane',
  ])

  // sync to allow preventDefault on keydown events
  paneActions.keydownOnPane.sync = true

  // sync to focus entry in same event loop cycle
  paneActions.toggleFocusMessage.sync = true
  paneActions.focusMessage.sync = true
  paneActions.moveMessageFocus.sync = true
  paneActions.focusEntry.sync = true
  paneActions.blurEntry.sync = true

  // sync to scroll in same render draw
  paneActions.scrollToEntry.sync = true
  paneActions.messageRenderFinished.sync = true
  paneActions.afterMessagesRendered.sync = true

  // sync so that message data changes take effect immediately (in relation to scrolling, etc)
  paneActions.setMessageData.sync = true

  var paneStore = Reflux.createStore({
    listenables: [
      paneActions,
      {chatChange: chat.store},
      {chatMessageReceived: chat.messageReceived},
    ],

    init: function() {
      this.state = {
        rootId: createOptions.rootId || '__root',
        focusedMessage: null,
        focusOwnNextMessage: false,
        entryText: '',
        entrySelectionStart: null,
        entrySelectionEnd: null,
        messageData: {},
      }

      this.messageData = new MessageData({
        focused: false,
        repliesExpanded: false,
        contentExpanded: false,
      })

      this.messageRenderFinished = _.debounce(paneActions.afterMessagesRendered, 0, {leading: true, trailing: false})
    },

    getInitialState: function() {
      return this.state
    },

    _set: function(data) {
      _.assign(this.state, data)
      this.trigger(this.state)
    },

    chatChange: function(state) {
      this.chatState = state
    },

    chatMessageReceived: function(message) {
      if (this.state.focusOwnNextMessage && message.get('_own') && message.get('parent') == '__root') {
        this.focusMessage(message.get('id'))
      }
    },

    sendMessage: function(content) {
      var parentId = this.state.focusedMessage
      actions.sendMessage(content, parentId)

      if (parentId) {
        this.setMessageData(parentId, {repliesExpanded: true})
      }

      if (!parentId || parentId == '__root') {
        this.state.focusOwnNextMessage = true
      }
    },

    focusMessage: function(messageId) {
      if (!this.chatState.nick) {
        return
      }

      messageId = messageId || this.state.rootId
      if (messageId == '__root') {
        messageId = null
      }

      if (messageId == this.state.focusedMessage) {
        return
      }

      // batch so that adding/removing focused ui doesn't cause scrolling
      React.addons.batchedUpdates(() => {
        if (this.state.focusedMessage) {
          this.setMessageData(this.state.focusedMessage, {focused: false})
        }
        if (messageId) {
          this.setMessageData(messageId, {focused: true})
        }
        this.state.focusedMessage = messageId
        this.trigger(this.state)

        require('react/lib/ReactUpdates').asap(() => {
          paneActions.scrollToEntry()
        })
      })
    },

    toggleFocusMessage: function(messageId, parentId) {
      var focusParent
      if (parentId == '__root') {
        parentId = null
        focusParent = this.state.focusedMessage == messageId
      } else {
        focusParent = this.state.focusedMessage != parentId
      }

      if (focusParent) {
        paneActions.focusMessage(parentId)
      } else {
        paneActions.focusMessage(messageId)
      }
    },

    revealMessage: function(messageId) {
      React.addons.batchedUpdates(() => {
        Immutable.Seq(this.chatState.messages.iterAncestorsOf(messageId))
          .forEach(ancestor => {
            this.setMessageData(ancestor.get('id'), {repliesExpanded: true})
          })
        this.setMessageData(messageId, {repliesExpanded: true})
      })
    },

    escape: function() {
      storeActions.deselectThread()
      paneActions.moveMessageFocus('top')
    },

    setMessageData: function(messageId, data) {
      this.messageData.set(messageId, data)
    },

    setEntryText: function(text, selectionStart, selectionEnd) {
      this.state.entryText = text
      this.state.entrySelectionStart = selectionStart
      this.state.entrySelectionEnd = selectionEnd
      // Note: no need to trigger here as nothing updates from this; this data is
      // used to persist entry state across focus changes.
    },

    openFocusedMessageInPane: function() {
      if (!this.state.focusedMessage) {
        return
      }
      storeActions.openThreadPane(this.state.focusedMessage)
    },
  })

  paneActions.store = paneStore
  paneActions.id = paneId
  paneActions.readOnly = Boolean(createOptions.readOnly)

  return paneActions
}
