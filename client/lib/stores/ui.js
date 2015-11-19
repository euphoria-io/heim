import _ from 'lodash'
import Reflux from 'reflux'
import ReactDOM from 'react-dom'
import Immutable from 'immutable'

import clamp from '../clamp'
import actions from '../actions'
import storage from './storage'
import chat from './chat'
import notification from './notification'
import MessageData from '../message-data'


const storeActions = Reflux.createActions([
  'keydownOnPage',
  'tabKeyCombo',
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
  'startMessageSelectionDrag',
  'finishMessageSelectionDrag',
  'openManagerToolbox',
  'closeManagerToolbox',
  'notificationsNoticeChoice',
  'dismissNotice',
])
_.extend(module.exports, storeActions)

// sync to allow entry to preventDefault keydown events
storeActions.keydownOnPage.sync = true
storeActions.tabKeyCombo.sync = true

// sync so UI mode changes don't flicker on load
storeActions.setUISize.sync = true

// sync so that UI pans start animating immediately
storeActions.panViewTo.sync = true

// temporarily sync while testing this commit
storeActions.focusPane.sync = true

const Pane = module.exports.Pane = class Pane {
  constructor(paneActions, store, id, readOnly) {
    this.store = store
    this.id = id
    this.readOnly = readOnly
    _.extend(this, paneActions)
  }
}

function createPaneStore(paneId, createOptions = {}) {
  const paneActions = Reflux.createActions([
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

  const paneStore = Reflux.createStore({
    listenables: [
      paneActions,
      {chatChange: chat.store},
      {chatMessageReceived: chat.messageReceived},
    ],

    init() {
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

    getInitialState() {
      return this.state
    },

    _set(data) {
      _.assign(this.state, data)
      this.trigger(this.state)
    },

    chatChange(state) {
      this.chatState = state
    },

    chatMessageReceived(message) {
      if (this.state.focusOwnNextMessage && message.get('_own') && message.get('parent') === '__root') {
        this.focusMessage(message.get('id'))
      }
    },

    sendMessage(content) {
      const parentId = this.state.focusedMessage
      actions.sendMessage(content, parentId)

      if (parentId) {
        this.setMessageData(parentId, {repliesExpanded: true})
      }

      if (!parentId || parentId === '__root') {
        this.state.focusOwnNextMessage = true
      }
    },

    focusMessage(messageId) {
      if (!this.chatState.nick) {
        return
      }

      let targetId = messageId || this.state.rootId
      if (targetId === '__root') {
        targetId = null
      }

      if (targetId === this.state.focusedMessage) {
        return
      }

      // batch so that adding/removing focused ui doesn't cause scrolling
      ReactDOM.unstable_batchedUpdates(() => {
        if (this.state.focusedMessage) {
          this.setMessageData(this.state.focusedMessage, {focused: false})
        }
        if (targetId) {
          this.setMessageData(targetId, {focused: true})
        }
        this.state.focusedMessage = targetId
        this.trigger(this.state)

        require('react/lib/ReactUpdates').asap(() => {
          paneActions.scrollToEntry()
        })
      })
    },

    toggleFocusMessage(messageId, parentId) {
      let focusParent
      if (parentId === '__root') {
        focusParent = this.state.focusedMessage === messageId
      } else {
        focusParent = this.state.focusedMessage !== parentId
      }

      if (focusParent) {
        paneActions.focusMessage(parentId)
      } else {
        paneActions.focusMessage(messageId)
      }
    },

    revealMessage(messageId) {
      ReactDOM.unstable_batchedUpdates(() => {
        Immutable.Seq(this.chatState.messages.iterAncestorsOf(messageId))
          .forEach(ancestor => {
            this.setMessageData(ancestor.get('id'), {repliesExpanded: true})
          })
        this.setMessageData(messageId, {repliesExpanded: true})
      })
    },

    escape() {
      storeActions.deselectThread()
      paneActions.moveMessageFocus('top')
    },

    setMessageData(messageId, data) {
      this.messageData.set(messageId, data)
    },

    setEntryText(text, selectionStart, selectionEnd) {
      this.state.entryText = text
      this.state.entrySelectionStart = selectionStart
      this.state.entrySelectionEnd = selectionEnd
      // Note: no need to trigger here as nothing updates from this; this data is
      // used to persist entry state across focus changes.
    },

    startEntryDrag() {
      this.state.draggingEntry = true
      this.trigger(this.state)
    },

    finishEntryDrag() {
      this.state.draggingEntry = false
      this.state.draggingEntryCommand = null
      this.trigger(this.state)
    },

    setEntryDragCommand(command) {
      if (command === this.state.draggingEntryCommand) {
        return
      }
      this.state.draggingEntryCommand = command
      this.trigger(this.state)
    },

    openFocusedMessageInPane() {
      if (!this.state.focusedMessage) {
        return
      }
      storeActions.openThreadPane(this.state.focusedMessage)
    },
  })

  return new Pane(paneActions, paneStore, paneId, Boolean(createOptions.readOnly))
}

const store = module.exports.store = Reflux.createStore({
  listenables: [
    actions,
    storeActions,
    {storageChange: storage.store},
    {chatChange: chat.store},
    {notificationChange: notification.store},
  ],

  init() {
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
      managerToolboxAnchorEl: null,
      draggingMessageSelection: false,
      draggingMessageSelectionToggle: null,
      notices: Immutable.OrderedSet(),
      notificationsNoticeDismissed: false,
    }

    this.threadData = new MessageData({selected: false})

    this._thawInfoDebounced = _.debounce(this._thawInfo, 1500)
  },

  getInitialState() {
    return this.state
  },

  storageChange(data) {
    if (!data) {
      return
    }
    this.state.infoPaneExpanded = _.get(data, ['room', this.chatState.roomName, 'infoPaneExpanded'], false)
    this.state.sidebarPaneExpanded = _.get(data, ['room', this.chatState.roomName, 'sidebarPaneExpanded'], true)
    this.state.notificationsNoticeDismissed = _.get(data, ['room', this.chatState.roomName, 'notificationsNoticeDismissed'], false)
    this._updateNotices()
    this.trigger(this.state)
  },

  chatChange(state) {
    this.chatState = state
    this._updateNotices()
    this.trigger(this.state)
  },

  notificationChange(state) {
    this.notificationState = state
  },

  _updateNotices() {
    const notifications = this.chatState.joined && this.chatState.nick && !this.state.notificationsNoticeDismissed
    if (notifications) {
      this.state.notices = this.state.notices.add('notifications')
    } else {
      this.state.notices = this.state.notices.delete('notifications')
    }
  },

  setUISize(width, height) {
    const thin = width < 650
    if (this.state.thin !== thin) {
      this.deselectThread()
    }
    this.state.thin = thin
    this.state.scrollEdgeSpace = height < 650 ? 50 : 156
    this.state.showTimestamps = width > 525
    this.trigger(this.state)
  },

  collapseInfoPane() {
    if (this.state.thin) {
      storeActions.panViewTo('main')
    } else {
      storage.setRoom(this.chatState.roomName, 'infoPaneExpanded', false)
    }
  },

  expandInfoPane() {
    if (this.state.thin) {
      storeActions.panViewTo('info')
    } else {
      storage.setRoom(this.chatState.roomName, 'infoPaneExpanded', true)
    }
  },

  toggleUserList() {
    if (this.state.thin) {
      storeActions.panViewTo(this.state.panPos === 'sidebar' ? 'main' : 'sidebar')
    } else {
      storage.setRoom(this.chatState.roomName, 'sidebarPaneExpanded', !this.state.sidebarPaneExpanded)
    }
  },

  freezeInfo() {
    this._thawInfoDebounced.cancel()
    if (!this.state.frozenThreadList || !this.state.frozenNotifications) {
      this.state.frozenThreadList = this.chatState.messages.threads.clone()
      this.state.frozenNotifications = this.notificationState.notifications
      this.trigger(this.state)
    }
  },

  thawInfo() {
    this._thawInfoDebounced()
  },

  _thawInfo() {
    if (this.state.threadPopupAnchorEl) {
      return
    }
    this.state.frozenThreadList = null
    this.state.frozenNotifications = null
    this.trigger(this.state)
  },

  _touchThreadPane(threadId) {
    const paneId = 'thread-' + threadId
    if (!this.state.panes.has(paneId)) {
      this.state.panes = this.state.panes.set(paneId, createPaneStore(paneId, {rootId: threadId}))
    }
    const pane = this.state.panes.get(paneId)
    pane.focusMessage(threadId)
    return pane
  },

  selectThread(id, el) {
    ReactDOM.unstable_batchedUpdates(() => {
      const pane = this._touchThreadPane(id)
      if (this.state.selectedThread && this.state.selectedThread !== id) {
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

  deselectThread() {
    if (!this.state.selectedThread) {
      return
    }
    ReactDOM.unstable_batchedUpdates(() => {
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

  openThreadPane(threadId) {
    ReactDOM.unstable_batchedUpdates(() => {
      const pane = this._touchThreadPane(threadId)
      this.state.visiblePanes = this.state.visiblePanes.add(pane.id)
      this.deselectThread()
      this.chatState.messages.mergeNodes(threadId, {_inPane: pane.id})
      this.focusPane(pane.id)
      this.trigger(this.state)
    })
  },

  popupToThreadPane() {
    this.openThreadPane(this.state.selectedThread)
  },

  closeThreadPane(threadId) {
    ReactDOM.unstable_batchedUpdates(() => {
      const paneId = 'thread-' + threadId
      this.focusPane('main')
      this.state.panes = this.state.panes.delete(paneId)
      this.state.visiblePanes = this.state.visiblePanes.remove(paneId)
      this.chatState.messages.mergeNodes(threadId, {_inPane: false})
      this.trigger(this.state)
    })
  },

  closeFocusedThreadPane() {
    if (!/^thread-/.test(this.state.focusedPane)) {
      return
    }
    this.closeThreadPane(this._focusedPane().store.state.rootId)
  },

  _focusedPane() {
    return this.state.panes.get(this.state.focusedPane)
  },

  focusPane(id) {
    if (this.state.focusedPane === id) {
      return
    }

    ReactDOM.unstable_batchedUpdates(() => {
      const lastFocused = this.state.focusedPane
      this.state.focusedPane = id
      this.trigger(this.state)

      require('react/lib/ReactUpdates').asap(() => {
        const lastFocusedPane = this.state.panes.get(lastFocused)
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

  _moveFocusedPane(delta) {
    const focusablePanes = this.state.visiblePanes
      .toKeyedSeq()
      .map(paneId => this.state.panes.get(paneId))
      .filterNot(pane => pane.readOnly)
      .cacheResult()

    let idx
    if (this.state.focusedPane === this.state.popupPane) {
      idx = -1
    } else {
      idx = focusablePanes.keySeq().indexOf(this.state.focusedPane)
    }
    idx = clamp(-1, idx + delta, focusablePanes.size - 1)

    if (idx === -1) {
      if (this.state.thin || !this.state.infoPaneExpanded) {
        storeActions.panViewTo('info')
      } else {
        this.onViewPan('info')
      }
    } else {
      storeActions.panViewTo('main')
      const paneId = focusablePanes.entrySeq().get(idx)[0]
      this.focusPane(paneId)
    }
  },

  focusLeftPane() {
    this._moveFocusedPane(-1)
  },

  focusRightPane() {
    this._moveFocusedPane(1)
  },

  focusEntry(character) {
    this._focusedPane().focusEntry(character)
  },

  onViewPan(target) {
    this.state.panPos = target
    this.trigger(this.state)
    if (this.state.thin) {
      if (target === 'info') {
        this.freezeInfo()
      } else {
        this.thawInfo()
      }
    } else {
      if (target === 'info') {
        if (!this.state.selectedThread) {
          storeActions.selectThreadInList(this.state.lastSelectedThread)
          return
        }
      } else {
        this.deselectThread()
      }
    }
  },

  keydownOnPage(ev) {
    if (Heim.tabPressed) {
      storeActions.tabKeyCombo(ev)
    } else {
      this._focusedPane().keydownOnPane(ev)
      // FIXME: this is a hack to detect whether a KeyboardActionHandler hasn't
      // triggered, and default to focusing the entry if no text is currently
      // selected (so we don't disrupt ctrl-c). this should be cleaned up with
      // an overhaul of the keyboard focus / combo handling code.
      if (!ev.isPropagationStopped() && uiwindow.getSelection().isCollapsed) {
        this.focusEntry()
      }
    }
  },

  gotoMessageInPane(messageId) {
    const parentPaneId = Immutable.Seq(this.chatState.messages.iterAncestorsOf(messageId))
      .map(ancestor => 'thread-' + ancestor.get('id'))
      .find(threadId => this.state.visiblePanes.has(threadId))

    const parentPane = this.state.panes.get(parentPaneId || 'main')

    ReactDOM.unstable_batchedUpdates(() => {
      parentPane.revealMessage(messageId)
      parentPane.focusMessage(messageId)
      parentPane.scrollToEntry()
    })
  },

  gotoPopupMessage() {
    const mainPane = this.state.panes.get('main')
    ReactDOM.unstable_batchedUpdates(() => {
      mainPane.revealMessage(this.state.selectedThread)
      mainPane.focusMessage(this.state.selectedThread)
      this.deselectThread()
    })
  },

  _createCustomPane(paneId, options) {
    const newPane = createPaneStore(paneId, options)
    this.state.panes = this.state.panes.set(paneId, newPane)
    this.trigger(this.state)
    return newPane
  },

  toggleManagerMode() {
    this.state.managerMode = !this.state.managerMode
    if (!this.state.managerMode) {
      chat.deselectAll()
      this.closeManagerToolbox()
    }
    this.trigger(this.state)
  },

  openManagerToolbox(anchorEl) {
    this.state.managerToolboxAnchorEl = anchorEl
    if (this.state.thin) {
      storeActions.panViewTo('main')
    }
    this.trigger(this.state)
  },

  closeManagerToolbox() {
    this.state.managerToolboxAnchorEl = null
    this.trigger(this.state)
  },

  startMessageSelectionDrag(toggleState) {
    this.state.draggingMessageSelection = true
    this.state.draggingMessageSelectionToggle = toggleState
    this.trigger(this.state)
  },

  finishMessageSelectionDrag() {
    this.state.draggingMessageSelection = false
    this.trigger(this.state)
  },

  notificationsNoticeChoice(choice) {
    notification.enablePopups()
    notification.setRoomNotificationMode(this.chatState.roomName, choice)
    this.dismissNotice('notifications')
  },

  dismissNotice(name) {
    if (name === 'notifications') {
      storage.setRoom(this.chatState.roomName, 'notificationsNoticeDismissed', true)
    }
  },
})

module.exports.createCustomPane = function createCustomPane(paneId, options) {
  return store._createCustomPane(paneId, options)
}
