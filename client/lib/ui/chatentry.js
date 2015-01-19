var _ = require('lodash')
var React = require('react/addons')
var Reflux = require('reflux')

var actions = require('../actions')


module.exports = React.createClass({
  displayName: 'ChatEntry',

  mixins: [
    require('react-immutable-render-mixin'),
    Reflux.connect(require('../stores/chat').store),
    Reflux.listenTo(actions.focusEntry, 'focus'),
  ],

  focus: function() {
    this.refs.input.getDOMNode().focus()
  },

  setNick: function(ev) {
    var input = this.refs.nick.getDOMNode()
    actions.setNick(input.value)
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
          return
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
          return
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
    if (!this.state.connected) {
      return
    }
    actions.sendMessage(text, this.state.focusedMessage)
    actions.setEntryText('')
    this.refs.input.getDOMNode().value = ''
  },

  onKeyDown: function(ev) {
    if (ev.shiftKey) {
      return
    }

    this.onTextChange(ev)

    var input = this.refs.input.getDOMNode()
    var length = input.value.length

    if (length) {
      if (ev.which == 13) {
        this.chatSend(input.value)
        ev.preventDefault()
      }
    } else {
      switch (ev.which) {
        case 37:
          this.chatMove('left')
          return
        case 27:  // ESC
        case 39:
          this.chatMove('right')
          return
      }
    }

    switch (ev.which) {
      case 38:
        this.chatMove('up')
        ev.preventDefault()
        break
      case 40:
        this.chatMove('down')
        ev.preventDefault()
        break
    }
  },

  previewNick: function() {
    var input = this.refs.nick.getDOMNode()
    this.setState({nickText: input.value})
  },

  onTextChange: function(ev) {
    actions.setEntryText(ev.target.value, ev.target.selectionStart, ev.target.selectionEnd)
  },

  componentDidMount: function() {
    this.refs.input.getDOMNode().setSelectionRange(this.state.entrySelectionStart, this.state.entrySelectionEnd)
  },

  render: function() {
    return (
      <form className="entry">
        <div className="nick-box">
          <div className="auto-size-container">
            <input className="nick" ref="nick" defaultValue={this.state.nick} onBlur={this.setNick} onChange={this.previewNick} />
            <span className="nick">{this.state.nickText || this.state.nick}</span>
          </div>
        </div>
        <input key="msg" ref="input" type="text" autoFocus defaultValue={this.state.entryText} onChange={this.onTextChange} onKeyDown={this.onKeyDown} onClick={this.onTextChange} />
      </form>
    )
  },

  componentWillUnmount: function() {
    // FIXME: hack to work around Reflux #156.
    this.replaceState = function() {}
  },
})
