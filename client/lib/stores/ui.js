var _ = require('lodash')
var Reflux = require('reflux')
var React = require('react/addons')
var Immutable = require('immutable')
var EventEmitter = require('eventemitter3')

var clamp = require('../clamp')
var actions = require('../actions')
var storage = require('./storage')
var chat = require('./chat')
var notification = require('./notification')
var MessageData = require('../message-data')


var storeActions = module.exports.actions = Reflux.createActions([
  'keydownOnPage',
  'focusEntry',
  'focusPane',
  'focusLeftPane',
  'focusRightPane',
  'collapseInfoPane',
  'expandInfoPane',
  'freezeInfo',
  'thawInfo',
  'showThreadPopup',
  'hideThreadPopup',
  'selectThreadInList',
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
      focusedPane: 'main',
      panes: Immutable.OrderedMap().withMutations(m => {
        m.set('popup', createPaneStore('popup'))
        m.set('main', createPaneStore('main'))
      }),
      infoPaneExpanded: false,
      frozenThreadList: null,
      frozenNotifications: null,
      threadPopupAnchorEl: null,
    }

    this.threadData = new MessageData({selected: false})

    this.state.panes.get('popup').escape.listen(this.hideThreadPopup)
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

  showThreadPopup: function(id, el) {
    React.addons.batchedUpdates(() => {
      var popupPane = this.state.panes.get('popup')
      this.freezeInfo()
      if (this.state.threadPopupRoot && this.state.threadPopupRoot != id) {
        this.threadData.set(this.state.threadPopupRoot, {selected: false})
      }
      this.threadData.set(id, {selected: true})
      this.state.threadPopupRoot = id
      this.state.threadPopupAnchorEl = el
      popupPane.store._reset({rootId: id})
      popupPane.focusMessage(id)
      this.focusPane('popup')
      this.trigger(this.state)
    })
  },

  hideThreadPopup: function() {
    var popupPane = this.state.panes.get('popup')
    this.thawInfo()
    if (!this.state.threadPopupRoot || popupPane.store.state.entryText.length) {
      return
    }
    this.threadData.set(this.state.threadPopupRoot, {selected: false})
    this.state.threadPopupAnchorEl = null
    this.focusPane('main')
    this.trigger(this.state)
  },

  openThreadPane: function(threadId) {
    React.addons.batchedUpdates(() => {
      var paneId = 'thread-' + threadId
      if (!this.state.panes.has(paneId)) {
        this.state.panes = this.state.panes.set(paneId, createPaneStore(paneId, {rootId: threadId}))
      }
      this.state.panes.get(paneId).focusMessage(threadId)
      this.hideThreadPopup()
      this.chatState.messages.mergeNodes(threadId, {_inPane: paneId})
      this.focusPane(paneId)
      this.trigger(this.state)
    })
  },

  popupToThreadPane: function() {
    this.openThreadPane(this.state.threadPopupRoot)
  },

  closeThreadPane: function(threadId) {
    React.addons.batchedUpdates(() => {
      var paneId = 'thread-' + threadId
      this.focusPane('main')
      this.state.panes = this.state.panes.delete(paneId)
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

  focusPane: function(id) {
    if (this.state.focusedPane == id) {
      return
    }

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
        this.state.panes.get(id).focusEntry()
      })
    })
  },

  _moveFocusedPane: function(delta) {
    var focusablePanes = this.state.panes.filterNot(pane => pane.readOnly)
    var idx = focusablePanes.keySeq().indexOf(this.state.focusedPane)
    idx = clamp(0, idx + delta, focusablePanes.size - 1)

    var paneId = focusablePanes.entrySeq().get(idx)[0]

    if (paneId == 'popup') {
      if (!this.state.threadPopupAnchorEl) {
        storeActions.selectThreadInList(this.state.threadPopupRoot)
        return
      }
    } else if (this.state.focusedPane == 'popup') {
      this.hideThreadPopup()
    }

    this.focusPane(paneId)
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

  keydownOnPage: function(ev) {
    this._focusedPane().keydownOnPane(ev)
  },

  gotoMessageInPane: function(messageId) {
    var parentPaneId = Immutable.Seq(this.chatState.messages.iterAncestorsOf(messageId))
      .map(ancestor => 'thread-' + ancestor.get('id'))
      .find(threadId => this.state.panes.has(threadId))

    var parentPane = this.state.panes.get(parentPaneId || 'main')

    React.addons.batchedUpdates(() => {
      parentPane.revealMessage(messageId)
      parentPane.focusMessage(messageId)
    })
  },

  gotoPopupMessage: function() {
    var mainPane = this.state.panes.get('main')
    React.addons.batchedUpdates(() => {
      this.hideThreadPopup()
      mainPane.revealMessage(this.state.threadPopupRoot)
      mainPane.focusMessage(this.state.threadPopupRoot)
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
      this._reset()

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

    _reset: function(options) {
      options = options || createOptions
      this.state = {
        rootId: options.rootId || '__root',
        focusedMessage: null,
        focusOwnNextMessage: false,
        entryText: '',
        entrySelectionStart: null,
        entrySelectionEnd: null,
        messageData: {},
      }
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
