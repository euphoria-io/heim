var _ = require('lodash')
var React = require('react')
var classNames = require('classnames')
var Reflux = require('reflux')

var isTextInput = require('../is-text-input')
var actions = require('../actions')
var chat = require('../stores/chat')
var ui = require('../stores/ui')
var activity = require('../stores/activity')
var Scroller = require('./scroller')
var Message = require('./message')
var MessageList = require('./message-list')
var ChatEntry = require('./chat-entry')
var NickEntry = require('./nick-entry')
var PasscodeEntry = require('./passcode-entry')


module.exports = React.createClass({
  displayName: 'ChatPane',

  mixins: [
    Reflux.ListenerMixin,
    Reflux.connect(ui.store, 'ui'),
    Reflux.connect(chat.store, 'chat'),
    Reflux.connect(activity.store, 'activity'),
    Reflux.listenTo(chat.logsReceived, 'scrollUpdatePosition'),
    Reflux.listenTo(activity.becameActive, 'onActive'),

    // when a new pane is added, all of the other panes get squished and
    // need to update their scroll position
    Reflux.listenTo(ui.popupToThreadPane, 'scrollUpdatePosition'),
  ],

  componentWillMount: function() {
    this._markSeen = _.debounce(this.markSeen, 250)
  },

  componentDidMount: function() {
    this.listenTo(this.props.pane.store, state => this.setState({'pane': state}))
    this.listenTo(this.props.pane.scrollToEntry, 'scrollToEntry')
    this.listenTo(this.props.pane.afterMessagesRendered, 'afterMessagesRendered')
    this.listenTo(this.props.pane.moveMessageFocus, 'moveMessageFocus')

    this.props.pane.scrollToEntry()
    this._markSeen()
  },

  getDefaultProps: function() {
    return {
      disabled: false,
      nodeId: '__root',
      showParent: false,
      showTimeStamps: false,
      showAllReplies: false,
    }
  },

  getInitialState: function() {
    return {pane: this.props.pane.store.getInitialState()}
  },

  onActive: function() {
    this._markSeen()
  },

  onScroll: function(userScrolled) {
    this._markSeen()

    if (!Heim.isTouch || !userScrolled) {
      return
    }

    var activeEl = uidocument.activeElement
    if (this.getDOMNode().contains(activeEl) && isTextInput(activeEl)) {
      activeEl.blur()
    }
  },

  markSeen: function() {
    if (!this.state.activity.active || this.props.disabled) {
      return
    }

    var scroller = this.refs.scroller
    if (!scroller) {
      // the pane was removed while we waited on debounce
      return
    }

    // it's important to use .line here instead of the parent (which contains
    // the replies), so that the nodes are non-overlapping and in visible order
    var messages = this.getDOMNode().querySelectorAll('.message-node > .line')
    if (!messages.length) {
      return
    }

    var scrollPos = scroller.getPosition()
    var guessIdx = Math.min(messages.length - 1, Math.floor(scrollPos * messages.length))

    var scrollerBox = this.refs.scroller.getDOMNode().getBoundingClientRect()
    var midPoint = (scrollerBox.bottom - scrollerBox.top) / 2
    var checkPos = function(el) {
      var box = el.getBoundingClientRect()
      if (box.bottom > scrollerBox.top && box.top < scrollerBox.bottom) {
        return true
      } else {
        return (box.bottom + box.top) / 2 - midPoint
      }
    }

    var ids = []

    var curIdx = guessIdx
    var guessPos = checkPos(messages[curIdx])
    if (guessPos !== true) {
      // the sign of the guess position tells us which direction to look for
      // onscreen messages
      var dir = -Math.sign(guessPos)
      while (messages[curIdx] && checkPos(messages[curIdx]) !== true) {
        curIdx += dir
      }
    }

    var startIdx = curIdx
    while (messages[curIdx] && checkPos(messages[curIdx]) === true) {
      ids.push(messages[curIdx].parentNode.dataset.messageId)
      curIdx++
    }

    curIdx = startIdx - 1
    while (messages[curIdx] && checkPos(messages[curIdx]) === true) {
      ids.push(messages[curIdx].parentNode.dataset.messageId)
      curIdx--
    }

    chat.markMessagesSeen(ids)
  },

  scrollToEntry: function() {
    this.refs.scroller.scrollToTarget()
  },

  afterMessagesRendered: function() {
    if (this.props.afterMessagesRendered) {
      this.props.afterMessagesRendered()
    }
    this.scrollUpdatePosition()
  },

  scrollUpdatePosition: function() {
    this.refs.scroller.update()
    this._markSeen()

    // if the entry has disappeared, reset message focus
    if (!this.getDOMNode().querySelector('.focus-target')) {
      this.props.pane.focusMessage()
    }
  },

  moveMessageFocus: function(dir) {
    // FIXME: quick'n'dirty hack. a real tree traversal in the store
    // would be more efficient and testable.
    var node = this.getDOMNode()
    var anchors = node.querySelectorAll('.focus-anchor, .focus-target')
    var idx = _.indexOf(anchors, node.querySelector('.focus-target'))
    if (idx == -1) {
      throw new Error('could not locate focus point in document')
    }

    var anchor
    switch (dir) {
      case 'up':
        if (idx === 0) {
          return
        }
        idx--
        anchor = anchors[idx]
        break
      case 'down':
        idx++
        anchor = anchors[idx]
        break
      case 'out':
        if (!this.state.pane.focusedMessage) {
          break
        }
        var parentId = this.state.chat.messages.get(this.state.pane.focusedMessage).get('parent')
        anchor = anchors[idx]
        while (anchor && anchor.dataset.messageId != parentId) {
          idx++
          anchor = anchors[idx]
        }
        break
      case 'top':
        break
    }

    React.addons.batchedUpdates(() => {
      this.props.pane.focusMessage(anchor && anchor.dataset.messageId)
      require('react/lib/ReactUpdates').asap(() => {
        this.props.pane.focusEntry()
      })
    })
  },

  onClick: function(ev) {
    if (!uiwindow.getSelection().isCollapsed || ev.target.nodeName == 'BUTTON' || ev.target.nodeName == 'A') {
      return
    }

    if (this.state.ui.focusedPane != this.props.pane.id) {
      ui.focusPane(this.props.pane.id)
      ev.stopPropagation()
    }
  },

  render: function() {
    var entryFocus = this.state.activity.windowFocused && this.state.chat.connected !== false && this.state.ui.focusedPane == this.props.pane.id

    // TODO: move this logic out of here
    var entry
    if (this.state.chat.authType == 'passcode' && this.state.chat.authState && this.state.chat.authState != 'trying-stored') {
      entry = <PasscodeEntry pane={this.props.pane} />
    } else if (this.state.chat.joined && !this.state.chat.nick && !this.state.chat.tentativeNick) {
      entry = <NickEntry pane={this.props.pane} />
    } else if (!this.state.pane.focusedMessage) {
      entry = <ChatEntry pane={this.props.pane} />
    }

    var MessageComponent = this.props.showParent ? Message : MessageList

    return (
      <div className={classNames('chat-pane', {'timestamps-visible': this.props.showTimeStamps})} onClickCapture={this.onClick}>
        <Scroller
          ref="scroller"
          target=".focus-target"
          edgeSpace={156}
          className="messages-container"
          onScrollbarSize={this.props.onScrollbarSize}
          onResize={this.onResize}
          onScroll={this.onScroll}
          onNearTop={this.state.pane.rootId == '__root' && actions.loadMoreLogs}
        >
          <div className="messages-content">
            <div className={classNames('messages', {'entry-focus': entryFocus})}>
              <MessageComponent key={this.state.pane.rootId} pane={this.props.pane} tree={this.state.chat.messages} nodeId={this.state.pane.rootId} showTimeStamps={this.props.showTimeStamps} showAllReplies={this.props.showAllReplies} roomSettings={this.state.chat.roomSettings} />
              {this.state.pane.rootId == '__root' && entry}
            </div>
          </div>
        </Scroller>
      </div>
    )
  },

  componentWillUnmount: function() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = function() {}
  },
})
