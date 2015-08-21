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
  'setUISize',
  'focusEntry',
  'focusPane',
  'focusLeftPane',
  'focusRightPane',
  'collapseInfoPane',
  'expandInfoPane',
  'toggleUserList',
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
  'globalMouseUp',
  'globalMouseMove',
  'toggleManagerMode',
])
_.extend(module.exports, storeActions)

// sync to allow entry to preventDefault keydown events
storeActions.keydownOnPage.sync = true

// sync so UI mode changes don't flicker on load
storeActions.setUISize.sync = true

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
      scrollEdgeSpace: 156,
      showTimestamps: true,
      focusedPane: 'main',
      panes: Immutable.Map({
        main: createPaneStore('main'),
      }),
      visiblePanes: Immutable.OrderedSet(['main']),
      popupPane: null,
      panPos: 'main',
      sidebarPaneExpanded: false,
      infoPaneExpanded: false,
      frozenThreadList: null,
      frozenNotifications: null,
      selectedThread: null,
      lastSelectedThread: null,
      threadPopupAnchorEl: null,
      managerMode: false,
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
    this.state.sidebarPaneExpanded = _.get(data, ['room', this.chatState.roomName, 'sidebarPaneExpanded'], false)
    this.trigger(this.state)
  },

  chatChange: function(state) {
    this.chatState = state
  },

  notificationChange: function(state) {
    this.notificationState = state
  },

  setUISize: function(width, height) {
    var thin = width < 650
    if (this.state.thin != thin) {
      this.deselectThread()
    }
    this.state.thin = thin
    this.state.scrollEdgeSpace = height < 650 ? 50 : 156
    this.state.showTimestamps = width > 525
    this.trigger(this.state)
  },

  collapseInfoPane: function() {
    if (this.state.thin) {
      storeActions.panViewTo('main')
    } else {
      storage.setRoom(this.chatState.roomName, 'infoPaneExpanded', false)
    }
  },

  expandInfoPane: function() {
    if (this.state.thin) {
      storeActions.panViewTo('info')
    } else {
      storage.setRoom(this.chatState.roomName, 'infoPaneExpanded', true)
    }
  },

  toggleUserList: function() {
    if (this.state.thin) {
      storeActions.panViewTo(this.state.panPos == 'sidebar' ? 'main' : 'sidebar')
    } else {
      storage.setRoom(this.chatState.roomName, 'sidebarPaneExpanded', !this.state.sidebarPaneExpanded)
    }
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
        this.focusPane(pane.id)
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
      if (!this.state.thin) {
        if (!this.state.popupPane) {
          return
        }
        this.thawInfo()
        this.state.threadPopupAnchorEl = null
        this.state.popupPane = null
      }
      storeActions.panViewTo('main')
      storeActions.focusPane('main')
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
        if (!Heim.isTouch) {
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

    var idx
    if (this.state.focusedPane == this.state.popupPane) {
      idx = -1
    } else {
      idx = focusablePanes.keySeq().indexOf(this.state.focusedPane)
    }
    idx = clamp(-1, idx + delta, focusablePanes.size - 1)

    if (idx == -1) {
      if (this.state.thin || !this.state.infoPaneExpanded) {
        storeActions.panViewTo('info')
      } else {
        this.onViewPan('info')
      }
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
    this.state.panPos = target
    this.trigger(this.state)
    if (this.state.thin) {
      if (target == 'info') {
        this.freezeInfo()
      } else {
        this.thawInfo()
      }
    } else {
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
      parentPane.scrollToEntry()
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

  toggleManagerMode: function() {
    this.state.managerMode = !this.state.managerMode
    this.trigger(this.state)
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
    'startEntryDrag',
    'finishEntryDrag',
    'setEntryDragCommand',
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
        draggingEntry: false,
        draggingEntryCommand: null,
        messageData: {},
      }

      this.messageData = new MessageData({
        focused: false,
        repliesExpanded: null,
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

    startEntryDrag: function() {
      this.state.draggingEntry = true
      this.trigger(this.state)
    },

    finishEntryDrag: function() {
      this.state.draggingEntry = false
      this.state.draggingEntryCommand = null
      this.trigger(this.state)
    },

    setEntryDragCommand: function(command) {
      if (command == this.state.draggingEntryCommand) {
        return
      }
      this.state.draggingEntryCommand = command
      this.trigger(this.state)
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
