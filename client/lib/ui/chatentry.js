var _ = require('lodash')
var React = require('react/addons')
var Reflux = require('reflux')

var actions = require('../actions')
var chat = require('../stores/chat')

module.exports = React.createClass({
  displayName: 'ChatEntry',

  mixins: [
    Reflux.connect(chat.store, 'chat'),
    Reflux.listenTo(actions.focusEntry, 'focus'),
    Reflux.listenTo(actions.keydownOnEntry, 'onKeyDown'),
  ],

  componentDidMount: function() {
    this.refs.input.getDOMNode().setSelectionRange(this.state.chat.entrySelectionStart, this.state.chat.entrySelectionEnd)
  },

  getInitialState: function() {
    return {
      nickText: null,
      nickFocused: false,
    }
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

  onNickChange: function(ev) {
    this.setState({nickText: ev.target.value})
  },

  onNickFocus: function(ev) {
    this.setState({nickText: ev.target.value, nickFocused: true})
  },

  onNickBlur: function(ev) {
    actions.setNick(ev.target.value)
    this.setState({nickText: null, nickFocused: false})
  },

  saveEntryState: function() {
    var input = this.refs.input.getDOMNode()
    actions.setEntryText(input.value, input.selectionStart, input.selectionEnd)
  },

  render: function() {
    var nick
    if (this.state.nickFocused) {
      nick = this.state.nickText
    } else {
      nick = this.state.chat.tentativeNick || this.state.chat.nick
    }

    return (
      <form className="entry">
        <div className="nick-box">
          <div className="auto-size-container">
            <input className="nick" ref="nick" value={nick} onFocus={this.onNickFocus} onBlur={this.onNickBlur} onChange={this.onNickChange} />
            <span className="nick">{nick}</span>
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
