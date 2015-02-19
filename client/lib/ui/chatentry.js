var _ = require('lodash')
var React = require('react/addons')
var Reflux = require('reflux')

var actions = require('../actions')
var chat = require('../stores/chat')

module.exports = React.createClass({
  displayName: 'ChatEntry',

  mixins: [
    require('react-immutable-render-mixin'),
    Reflux.connect(chat.store, 'chat'),
    Reflux.listenTo(chat.store, 'onNickReply', 'onNickReply'),
    Reflux.listenTo(actions.focusEntry, 'focus'),
    Reflux.listenTo(actions.keydownOnEntry, 'onKeyDown'),
  ],

  componentDidMount: function() {
    this.refs.input.getDOMNode().setSelectionRange(this.state.chat.entrySelectionStart, this.state.chat.entrySelectionEnd)
    this._nickInFlight = false
  },

  focus: function(withChar) {
    var node = this.refs.input.getDOMNode()
    if (withChar) {
      node.value += withChar
      this.saveEntryState()
    }
    node.focus()
    actions.scrollToEntry()
  },

  setNick: function(ev) {
    var input = this.refs.nick.getDOMNode()
    actions.setNick(input.value)
    this._nickInFlight = true
    ev.preventDefault()
  },

  chatMove: function(dir) {
    // FIXME: quick'n'dirty hack. a real tree traversal in the store
    // would be more efficient and testable.
    var elems = document.querySelectorAll('.reply-anchor, .entry')
    var idx = _.indexOf(elems, this.getDOMNode())
    if (idx == -1) {
      throw new Error('could not locate entry in document')
    }

    var target
    switch (dir) {
      case 'up':
        if (idx === 0) {
          // at beginning
          target = elems[idx + 1].parentNode
          break
        }
        var steps = 0
        do {
          // find prev leaf
          idx--
          target = elems[idx]
          target = target && target.parentNode
          steps++
        } while (target.querySelectorAll('.replies').length)
        if (steps > 1) {
          // if we descended deeply, focus parent of leaf
          idx++
        }
        target = elems[idx]
        target = target && target.parentNode
        break
      case 'down':
        if (idx == elems.length - 1) {
          // at end
          target = elems[idx].parentNode
          break
        }
        idx++
        target = elems[idx]
        target = target && target.parentNode
        if (!target.querySelectorAll('.replies .message-node').length) {
          // last focused was a leaf
          idx++
          target = elems[idx]
          target = target && target.parentNode
        } else {
          // find next leaf
          do {
            idx++
            target = elems[idx]
            target = target && target.parentNode
          } while (target && target.querySelectorAll('.replies').length)
        }
        break
      case 'left':
        target = elems[idx]
        target = target && target.parentNode
        target = target && target.parentNode
        target = target && target.parentNode
        target = target && target.parentNode
        break
      case 'right':
        target = null
        break
    }
    actions.focusMessage(target && target.dataset.messageId)
  },

  chatSend: function(text) {
    if (!this.state.chat.connected) {
      return
    }
    actions.sendMessage(text, this.state.chat.focusedMessage)
    actions.setEntryText('')
    this.refs.input.getDOMNode().value = ''
  },

  onKeyDown: function(ev) {
    if (ev.shiftKey) {
      return
    }

    this.saveEntryState()

    var input = this.refs.input.getDOMNode()
    var length = input.value.length

    if (length) {
      if (ev.key == 'Enter') {
        this.chatSend(input.value)
        ev.preventDefault()
      }
    } else {
      switch (ev.key) {
        case 'ArrowLeft':
          this.chatMove('left')
          return
        case 'ArrowRight':
          this.chatMove('right')
          return
      }
    }

    switch (ev.key) {
      case 'Escape':
        this.chatMove('right')
        break
      case 'ArrowUp':
        this.chatMove('up')
        ev.preventDefault()
        break
      case 'ArrowDown':
        this.chatMove('down')
        ev.preventDefault()
        break
    }
  },

  previewNick: function() {
    var input = this.refs.nick.getDOMNode()
    this.setState({nickText: input.value})
  },

  saveEntryState: function() {
    var input = this.refs.input.getDOMNode()
    actions.setEntryText(input.value, input.selectionStart, input.selectionEnd)
  },

  onNickReply: function(chatState) {
    if (!chatState.nickInFlight && (this._nickInFlight || !this.state.nickText)) {
      this._nickInFlight = false
      if (!chatState.nickRejected) {
        this.setState({nickText: chatState.confirmedNick})
      }
    }
  },

  render: function() {
    return (
      <form className="entry">
        <div className="nick-box">
          <div className="auto-size-container">
            <input className="nick" ref="nick" value={this.state.nickText || this.state.chat.nick} onBlur={this.setNick} onChange={this.previewNick} />
            <span className="nick">{this.state.nickText || this.state.chat.nick}</span>
          </div>
        </div>
        <input key="msg" ref="input" type="text" autoFocus defaultValue={this.state.chat.entryText} onChange={this.saveEntryState} onKeyDown={this.onKeyDown} onClick={this.saveEntryState} onFocus={actions.scrollToEntry} onKeyPress={actions.scrollToEntry} />
      </form>
    )
  },

  componentWillUnmount: function() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = function() {}
  },
})
