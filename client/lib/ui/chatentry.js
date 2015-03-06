var _ = require('lodash')
var React = require('react/addons')
var Reflux = require('reflux')

var actions = require('../actions')
var chat = require('../stores/chat')

module.exports = React.createClass({
  displayName: 'ChatEntry',

  mixins: [
    require('./entrymixin'),
    Reflux.connect(chat.store, 'chat'),
    Reflux.listenTo(actions.focusEntry, 'focus'),
    Reflux.listenTo(actions.keydownOnEntry, 'onKeyDown'),
  ],

  componentWillMount: function() {
    this.setState({empty: !this.state.chat.entryText})
  },

  componentDidMount: function() {
    this.refs.input.getDOMNode().setSelectionRange(this.state.chat.entrySelectionStart, this.state.chat.entrySelectionEnd)
  },

  getInitialState: function() {
    return {
      nickText: null,
      nickFocused: false,
      empty: true,
    }
  },

  chatMove: function(dir) {
    // FIXME: quick'n'dirty hack. a real tree traversal in the store
    // would be more efficient and testable.
    var elems = uidocument.querySelectorAll('.reply-anchor, .entry')
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

  chatSend: function(ev) {
    var input = this.refs.input.getDOMNode()

    ev.preventDefault()

    if (!this.state.chat.connected) {
      return
    }

    if (!input.value.length) {
      return
    }
    actions.sendMessage(input.value, this.state.chat.focusedMessage)
    actions.setEntryText('')
    input.value = ''
    this.setState({empty: true})

    if (Heim.isAndroid) {
      // Emptying the input value doesn't reset the Android keyboard state.
      // This seems to work without causing a flicker.
      input.blur()
      input.focus()
    }
  },

  onKeyDown: function(ev) {
    if (ev.shiftKey) {
      return
    }

    var input = this.refs.input.getDOMNode()
    var length = input.value.length

    if (ev.target != input && this.proxyKeyDown(ev)) {
      return
    }

    this.saveEntryState()

    if (ev.key == 'Enter') {
      this.chatSend(ev)
      return
    }

    if (!length) {
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
    this.setState({empty: !input.value.length})
  },

  render: function() {
    var nick
    if (this.state.nickFocused) {
      nick = this.state.nickText
    } else {
      nick = this.state.chat.tentativeNick || this.state.chat.nick

      if (nick === null) {
        nick = 'loading...'
      }
    }

    return (
      <form className="entry" onSubmit={ev => ev.preventDefault()}>
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
})
