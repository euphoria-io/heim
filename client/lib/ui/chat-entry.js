var _ = require('lodash')
var React = require('react/addons')
var Reflux = require('reflux')

var actions = require('../actions')
var chat = require('../stores/chat')
var hueHash = require('../hue-hash')

module.exports = React.createClass({
  displayName: 'ChatEntry',

  mixins: [
    require('./entry-mixin'),
    Reflux.connect(chat.store, 'chat'),
    Reflux.listenTo(actions.focusEntry, 'focus'),
    Reflux.listenTo(actions.keydownOnEntry, 'onKeyDown'),
  ],

  componentWillMount: function() {
    this.setState({empty: !this.state.chat.entryText})
  },

  componentDidMount: function() {
    this.refs.input.getDOMNode().setSelectionRange(this.state.chat.entrySelectionStart, this.state.chat.entrySelectionEnd)
    this.autoSize(true)
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

    if (!/\n/.test(input.value)) {
      switch (ev.key) {
        case 'ArrowUp':
          this.chatMove('up')
          ev.preventDefault()
          return
        case 'ArrowDown':
          this.chatMove('down')
          ev.preventDefault()
          return
      }
    }

    switch (ev.key) {
      case 'Escape':
        this.chatMove('right')
        break
      case 'Tab':
        this.complete()
        ev.preventDefault()
        break
    }
  },

  complete: function() {
    var input = this.refs.input.getDOMNode()
    var text = input.value
    var charRe = /\S/

    var wordEnd = input.selectionStart
    if (wordEnd < 1) {
      return
    }

    for (var wordStart = wordEnd - 1; wordStart >= 0; wordStart--) {
      if (!charRe.test(text[wordStart])) {
        break
      }
    }
    wordStart++
    if (text[wordStart] == '@') {
      wordStart++
    }

    for (; wordEnd < text.length; wordEnd++) {
      if (!charRe.test(text[wordEnd])) {
        break
      }
    }

    if (wordStart >= wordEnd) {
      return
    }

    // FIXME: replace this with a fast Trie implementation
    var word = hueHash.stripSpaces(text.substring(wordStart, wordEnd)).toLowerCase()
    var match = this.state.chat.who
      .toSeq()
      .map(user => hueHash.stripSpaces(user.get('name', '')))
      .filter(Boolean)
      .map(name => [name.toLowerCase().lastIndexOf(word), name])
      .filter(entry => entry[0] > -1)
      .sort()
      .first()

    if (!match) {
      return
    }
    var completed = (text[wordStart - 1] != '@' ? '@' : '') + match[1]
    input.value = input.value.substring(0, wordStart) + completed + input.value.substring(wordEnd)
    this.saveEntryState()
  },

  onNickChange: function(ev) {
    this.setState({nickText: ev.target.value})
  },

  onNickKeyDown: function(ev) {
    var input = this.refs.input.getDOMNode()
    if (ev.key == 'Enter') {
      // Delay focus event to avoid double key insertion in Chrome.
      setImmediate(function() {
        input.focus()
      })
    } else if (ev.key == 'Escape') {
      this.setState({nickText: this.state.chat.nick}, function() {
        input.focus()
      })
    }
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
            <input className="nick" ref="nick" value={nick} onFocus={this.onNickFocus} onBlur={this.onNickBlur} onChange={this.onNickChange} onKeyDown={this.onNickKeyDown} />
            <span className="nick">{nick}</span>
          </div>
        </div>
        <textarea key="msg" ref="input" autoFocus defaultValue={this.state.chat.entryText} onChange={this.saveEntryState} onKeyDown={this.onKeyDown} onClick={this.saveEntryState} onFocus={actions.scrollToEntry} />
        <textarea key="measure" ref="measure" className="measure" />
      </form>
    )
  },

  autoSize: function(force) {
    var input = this.refs.input.getDOMNode()
    var measure = this.refs.measure.getDOMNode()
    if (force || input.value != this.state.chat.entryText) {
      measure.style.width = input.offsetWidth + 'px'
      measure.value = input.value
      input.style.height = measure.scrollHeight + 'px'
    }
  },

  componentDidUpdate: function() {
    this.autoSize()
  },
})
