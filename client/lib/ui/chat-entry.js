var React = require('react/addons')
var Reflux = require('reflux')

var actions = require('../actions')
var chat = require('../stores/chat')
var hueHash = require('../hue-hash')
var KeyboardActionHandler = require('./keyboard-action-handler')
var EntryDragHandle = require('./entry-drag-handle')

module.exports = React.createClass({
  displayName: 'ChatEntry',

  mixins: [
    require('./entry-mixin'),
    Reflux.ListenerMixin,
    Reflux.connect(chat.store, 'chat'),
  ],

  componentWillMount: function() {
    this.setState({empty: !this.state.chat.entryText})
  },

  componentDidMount: function() {
    this.listenTo(this.props.pane.store, state => this.setState({'pane': state}))
    this.listenTo(this.props.pane.focusEntry, 'focus')
    this.listenTo(this.props.pane.blurEntry, 'blur')
    var input = this.refs.input.getDOMNode()
    input.value = this.state.pane.entryText
    // in Chrome, it appears that setting the selection range can focus the
    // input without changing document.activeElement (!)
    if (this.state.pane.entrySelectionStart && this.state.pane.entrySelectionEnd) {
      input.setSelectionRange(this.state.pane.entrySelectionStart, this.state.pane.entrySelectionEnd)
    }
    this.autoSize(true)
  },

  getInitialState: function() {
    return {
      pane: this.props.pane.store.getInitialState(),
      nickText: null,
      nickFocused: false,
      empty: true,
    }
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
    this.props.pane.sendMessage(input.value)
    this.props.pane.setEntryText('')
    input.value = ''
    this.setState({empty: true})

    if (Heim.isAndroid) {
      // Emptying the input value doesn't reset the Android keyboard state.
      // This seems to work without causing a flicker.
      input.blur()
      input.focus()
    }

    this.props.pane.scrollToEntry()
  },

  isEmpty: function() {
    return this.refs.input.getDOMNode().value.length === 0
  },

  isMultiline: function() {
    return /\n/.test(this.refs.input.getDOMNode().value)
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
      ev.stopPropagation()
    } else if (ev.key == 'Escape') {
      this.setState({nickText: this.state.chat.nick}, function() {
        input.focus()
      })
      ev.stopPropagation()
    } else if (/^Arrow/.test(ev.key) || ev.key == 'Tab' || ev.key == 'Backspace') {
      // don't let the keyboard action handler react to these
      ev.stopPropagation()
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
    this.props.pane.setEntryText(input.value, input.selectionStart, input.selectionEnd)
    this.setState({empty: !input.value.length})
  },

  onChange: function(ev) {
    this.saveEntryState()
    if (this.props.onChange) {
      this.props.onChange(ev)
    }
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
      <KeyboardActionHandler listenTo={this.props.pane.keydownOnPane} keys={{
        ArrowLeft: () => this.isEmpty() && this.props.pane.moveMessageFocus('out'),
        ArrowRight: () => this.isEmpty() && this.props.pane.moveMessageFocus('top'),
        ArrowUp: () => !this.isMultiline() && this.props.pane.moveMessageFocus('up'),
        ArrowDown: () => !this.isMultiline() && this.props.pane.moveMessageFocus('down'),
        Escape: () => this.props.pane.escape(),
        Enter: this.chatSend,
        TabEnter: this.props.pane.openFocusedMessageInPane,
        Backspace: this.proxyKeyDown,
        Tab: this.complete,
      }}>
        <form className="entry focus-target" onSubmit={ev => ev.preventDefault()}>
          <div className="nick-box">
            <div className="auto-size-container">
              <input className="nick" ref="nick" value={nick} onFocus={this.onNickFocus} onBlur={this.onNickBlur} onChange={this.onNickChange} onKeyDown={this.onNickKeyDown} />
              <span className="nick">{nick}</span>
            </div>
          </div>
          <textarea key="msg" ref="input" className="entry-text" onChange={this.onChange} onKeyDown={this.saveEntryState} onClick={this.saveEntryState} />
          <textarea key="measure" ref="measure" className="measure" />
          <EntryDragHandle pane={this.props.pane} />
        </form>
      </KeyboardActionHandler>
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
