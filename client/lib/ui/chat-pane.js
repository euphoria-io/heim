import _ from 'lodash'
import React from 'react'
import classNames from 'classnames'
import Reflux from 'reflux'

import isTextInput from '../is-text-input'
import actions from '../actions'
import chat from '../stores/chat'
import ui, {Pane} from '../stores/ui'
import activity from '../stores/activity'
import Scroller from './scroller'
import Message from './message'
import MessageList from './message-list'
import ChatEntry from './chat-entry'
import NickEntry from './nick-entry'
import PasscodeEntry from './passcode-entry'
import messageCopyFormatter from './message-copy-formatter'


function boxMiddle(el) {
  if (_.isNumber(el)) {
    return el
  }
  const box = el.getBoundingClientRect()
  return box.top + box.height / 2
}

function closestIdx(array, value, iterator) {
  let idx = _.sortedIndex(array, value, iterator)
  // reminder: sortedIndex can return an index after the last index if
  // the element belongs at the end of the array.
  if (idx > 0 && idx < array.length && value - iterator(array[idx - 1]) < iterator(array[idx]) - value) {
    idx--
  }
  return idx
}

export default React.createClass({
  displayName: 'ChatPane',

  propTypes: {
    disabled: React.PropTypes.bool,
    nodeId: React.PropTypes.string,
    afterRender: React.PropTypes.func,
    onScrollbarSize: React.PropTypes.func,
    showParent: React.PropTypes.bool,
    showTimeStamps: React.PropTypes.bool,
    showAllReplies: React.PropTypes.bool,
    pane: React.PropTypes.instanceOf(Pane).isRequired,
  },

  mixins: [
    Reflux.ListenerMixin,
    Reflux.connect(ui.store, 'ui'),
    Reflux.connect(chat.store, 'chat'),
    Reflux.connect(activity.store, 'activity'),
    Reflux.listenTo(activity.becameActive, 'onActive'),

    // scroll when page size changes
    Reflux.listenTo(ui.setUISize, 'scrollUpdatePosition'),

    // when a new pane is added, all of the other panes get squished and
    // need to update their scroll position
    Reflux.listenTo(ui.popupToThreadPane, 'scrollUpdatePosition'),
  ],

  getDefaultProps() {
    return {
      disabled: false,
      showParent: false,
      showTimeStamps: false,
      showAllReplies: false,
    }
  },

  getInitialState() {
    return {pane: this.props.pane.store.getInitialState()}
  },

  componentWillMount() {
    this._dragEl = null
    this._dragMatch = null
    this._dragPos = null
    this._dragInterval = null
    this._markSeen = _.debounce(this.markSeen, 250)
  },

  componentDidMount() {
    this.listenTo(this.props.pane.store, state => this.setState({'pane': state}))
    this.listenTo(this.props.pane.scrollToEntry, 'scrollToEntry')
    this.listenTo(this.props.pane.afterMessagesRendered, 'scrollUpdatePosition')
    this.listenTo(this.props.pane.moveMessageFocus, 'moveMessageFocus')

    if (!this.isTouch) {
      this.listenTo(ui.globalMouseUp, 'onMessageMouseUp')
      this.listenTo(ui.globalMouseMove, 'onMessageMouseMove')
    }

    this.scrollToEntry({immediate: true})
    this._markSeen()
  },

  componentDidUpdate() {
    if (this.state.pane.draggingEntry && !this._dragInterval) {
      this._dragInterval = setInterval(this.onDragUpdate, 1000 / 10)
    } else if (!this.state.pane.draggingEntry && this._dragInterval) {
      clearInterval(this.onDragUpdate)
      this._dragInterval = null
    }

    this.scrollUpdatePosition()
  },

  componentWillUnmount() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = () => {}
  },

  onActive() {
    this._markSeen()
  },

  onScroll(userScrolled) {
    this._markSeen()

    if (!Heim.isTouch || !userScrolled) {
      return
    }

    const activeEl = uidocument.activeElement
    if (this.getDOMNode().contains(activeEl) && isTextInput(activeEl)) {
      activeEl.blur()
    }
  },

  // since the entry buttons are transient within the context of changing the
  // entry focus, we'll handle the dragging events and state where they bubble
  // up here.

  onMessageMouseDown(ev) {
    if (ev.button === 0 && ev.target.classList.contains('drag-handle')) {
      this._dragMatch = {button: ev.button}
      this.props.pane.startEntryDrag()
    }
  },

  onMessageMouseUp(ev) {
    if (!this.state.pane.draggingEntry) {
      return
    }
    if (_.isMatch(ev, this._dragMatch)) {
      this._finishDrag()
    }
  },

  onMessageMouseMove(ev) {
    if (!this.state.pane.draggingEntry) {
      return
    }
    this._dragPos = {
      x: ev.clientX,
      y: ev.clientY,
    }
  },

  onMessageTouchStart(ev) {
    if (ev.target.classList.contains('drag-handle')) {
      ev.preventDefault()
      ev.stopPropagation()
      // touch events originate from the original target. when the entry gets
      // removed from the page, touch events will stop bubbling from it, so we
      // need to subscribe directly.
      // http://bl.ocks.org/mbostock/770ae19ca830a4ce87f5
      ev.target.addEventListener('touchend', this.onMessageTouchEnd, false)
      ev.target.addEventListener('touchcancel', this.onMessageTouchEnd, false)
      ev.target.addEventListener('touchmove', this.onMessageTouchMove, false)
      this._dragEl = ev.target  // prevent Mobile Safari from garbage collecting our touch event emitter
      this._dragMatch = {identifier: ev.targetTouches[0].identifier}
      this.props.pane.startEntryDrag()
    }
  },

  onMessageTouchEnd(ev) {
    ev.target.removeEventListener('touchend', this.onMessageTouchEnd, false)
    ev.target.removeEventListener('touchcancel', this.onMessageTouchEnd, false)
    ev.target.removeEventListener('touchmove', this.onMessageTouchMove, false)
    if (!_.find(ev.touches, this._dragMatch)) {
      this._finishDrag()
    }
  },

  onMessageTouchMove(ev) {
    if (!this.state.pane.draggingEntry) {
      return
    }
    const touch = _.find(ev.touches, this._dragMatch)
    if (!touch) {
      return
    }
    ev.preventDefault()
    this._dragPos = {
      x: touch.clientX,
      y: touch.clientY,
    }
  },

  onDragUpdate() {
    const pos = this._dragPos
    if (pos) {
      this.focusMessageFromPos(pos.y)

      const over = uidocument.elementFromPoint(pos.x, pos.y)
      if (over && over.classList.contains('jump-to-bottom')) {
        this.props.pane.setEntryDragCommand('to-bottom')
      } else {
        this.props.pane.setEntryDragCommand(null)
      }
    }
  },

  onClick(ev) {
    if (!uiwindow.getSelection().isCollapsed || ev.target.nodeName === 'BUTTON' || ev.target.nodeName === 'A') {
      return
    }

    if (this.state.ui.focusedPane !== this.props.pane.id) {
      ui.focusPane(this.props.pane.id)
      ev.stopPropagation()
    }
  },

  markSeen() {
    if (!this.state.activity.active || this.props.disabled) {
      return
    }

    const scroller = this.refs.scroller
    if (!scroller) {
      // the pane was removed while we waited on debounce
      return
    }

    // it's important to use .line here instead of the parent (which contains
    // the replies), so that the nodes are non-overlapping and in visible order
    const messages = this.getDOMNode().querySelectorAll('.message-node > .line')
    if (!messages.length) {
      return
    }

    const scrollPos = scroller.getPosition()
    if (scrollPos === false) {
      // styles not loaded yet
      return
    }

    const guessIdx = Math.min(messages.length - 1, Math.floor(scrollPos * messages.length))

    const scrollerBox = this.refs.scroller.getDOMNode().getBoundingClientRect()
    const midPoint = (scrollerBox.bottom - scrollerBox.top) / 2
    const checkPos = el => {
      const box = el.getBoundingClientRect()
      if (box.bottom > scrollerBox.top && box.top < scrollerBox.bottom) {
        return true
      }
      return (box.bottom + box.top) / 2 - midPoint
    }

    const ids = []

    let curIdx = guessIdx
    const guessPos = checkPos(messages[curIdx])
    if (guessPos !== true) {
      // the sign of the guess position tells us which direction to look for
      // onscreen messages
      const dir = -Math.sign(guessPos)
      while (messages[curIdx] && checkPos(messages[curIdx]) !== true) {
        curIdx += dir
      }
    }

    const startIdx = curIdx
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

  scrollToEntry(options) {
    this.refs.scroller.scrollToTarget(options)
  },

  scrollUpdatePosition() {
    if (!this.refs.scroller) {
      // we've been unmounted as a result of setUISize but the listener has not
      // been removed yet.
      return
    }

    this.refs.scroller.update()
    this._markSeen()

    if (this.props.afterRender) {
      this.props.afterRender()
    }

    // if the entry has disappeared, reset message focus
    if (!this.getDOMNode().querySelector('.focus-target')) {
      this.props.pane.focusMessage()
    }
  },

  moveMessageFocus(dir) {
    // FIXME: quick'n'dirty hack. a real tree traversal in the store
    // would be more efficient and testable.
    const node = this.getDOMNode()
    const anchors = node.querySelectorAll('.focus-anchor, .focus-target')
    let idx = _.indexOf(anchors, node.querySelector('.focus-target'))
    if (idx === -1) {
      throw new Error('could not locate focus point in document')
    }

    let anchor
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
      const parentId = this.state.chat.messages.get(this.state.pane.focusedMessage).get('parent')
      anchor = anchors[idx]
      while (anchor && anchor.dataset.messageId !== parentId) {
        idx++
        anchor = anchors[idx]
      }
      break
    case 'top':
      break
    // no default
    }

    React.addons.batchedUpdates(() => {
      this.props.pane.focusMessage(anchor && anchor.dataset.messageId)
      if (!Heim.isTouch) {
        require('react/lib/ReactUpdates').asap(() => {
          this.props.pane.focusEntry()
        })
      }
    })
  },

  _finishDrag() {
    if (this.state.pane.draggingEntryCommand === 'to-bottom') {
      this.props.pane.focusMessage(null)
    }

    this._dragEl = null
    this._dragMatch = null
    this._dragPos = null
    this.props.pane.finishEntryDrag()

    if (!Heim.isTouch) {
      this.props.pane.focusEntry()
    }
  },

  focusMessageFromPos(yPos) {
    const node = this.getDOMNode()
    const anchorNodes = node.querySelectorAll('.focus-anchor, .focus-target')

    const messagesEl = this.refs.messages.getDOMNode()
    const endPos = messagesEl.getBoundingClientRect().bottom
    const anchors = _.toArray(anchorNodes)
    anchors.push(endPos)

    let idx = closestIdx(anchors, yPos, boxMiddle)

    if (idx >= anchors.length - 1) {
      this.props.pane.focusMessage(null)
      return
    }

    // weight the current position towards nearby focus anchors
    let totalPos = yPos
    let count = 1
    for (let i = -3; i <= 3; i++) {
      const a = anchors[idx + i]
      if (!a) {
        continue
      }

      const pos = boxMiddle(a)
      const factor = 2 - Math.pow(Math.abs(yPos - pos) / 40, 2)
      if (factor > 0) {
        totalPos += pos * factor
        count += factor
      }
    }

    const weighted = totalPos / count
    idx = closestIdx(anchors, weighted, boxMiddle)

    const choiceId = anchors[idx].dataset.messageId
    // check if already focused, force scroll if necessary
    if (!choiceId || choiceId === this.state.pane.focusedMessage) {
      const scrollPos = this.refs.scroller.getPosition()
      const scrollEdgeSpace = this.state.ui.scrollEdgeSpace
      if (yPos < scrollEdgeSpace && scrollPos > 0) {
        this.moveMessageFocus('up')
      } else if (yPos >= node.getBoundingClientRect().bottom - scrollEdgeSpace && scrollPos < 1) {
        this.moveMessageFocus('down')
      }
    } else {
      this.props.pane.focusMessage(choiceId)
    }
  },

  render() {
    const entryFocus = this.state.activity.windowFocused && this.state.chat.connected !== false && this.state.ui.focusedPane === this.props.pane.id

    // TODO: move this logic out of here
    let entry
    if (this.state.chat.authType === 'passcode' && this.state.chat.authState && this.state.chat.authState !== 'trying-stored') {
      entry = <PasscodeEntry pane={this.props.pane} />
    } else if (this.state.chat.joined && !this.state.chat.nick && !this.state.chat.tentativeNick) {
      entry = <NickEntry pane={this.props.pane} />
    } else if (!this.state.pane.focusedMessage) {
      entry = <ChatEntry pane={this.props.pane} />
    }

    const MessageComponent = this.props.showParent ? Message : MessageList

    let messageDragEvents
    if (Heim.isTouch) {
      messageDragEvents = {
        onTouchStart: this.onMessageTouchStart,
      }
    } else {
      messageDragEvents = {
        onMouseDown: this.onMessageMouseDown,
      }
    }

    return (
      <div className={classNames('chat-pane', {'timestamps-visible': this.props.showTimeStamps})} onClickCapture={this.onClick} onCopy={messageCopyFormatter}>
        <Scroller
          ref="scroller"
          target=".focus-target"
          edgeSpace={this.state.ui.scrollEdgeSpace}
          className="messages-container"
          onScrollbarSize={this.props.onScrollbarSize}
          onScroll={this.onScroll}
          onNearTop={this.state.pane.rootId === '__root' ? actions.loadMoreLogs : null}
        >
          <div className="messages-content">
            <div ref="messages" className={classNames('messages', {'entry-focus': entryFocus, 'entry-dragging': this.state.pane.draggingEntry})} {...messageDragEvents}>
              {this.state.chat.authState && this.state.chat.authState !== 'trying-stored' && <div className="hatch-shade fill" />}
              <MessageComponent key={this.state.pane.rootId} pane={this.props.pane} tree={this.state.chat.messages} nodeId={this.state.pane.rootId} showTimeStamps={this.props.showTimeStamps} showAllReplies={this.props.showAllReplies} roomSettings={this.state.chat.roomSettings} />
              {this.state.pane.rootId === '__root' && entry}
            </div>
          </div>
        </Scroller>
      </div>
    )
  },
})
