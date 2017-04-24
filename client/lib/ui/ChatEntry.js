import React from 'react'
import classNames from 'classnames'
import Reflux from 'reflux'

import actions from '../actions'
import { Pane } from '../stores/ui'
import chat from '../stores/chat'
import mention from '../mention'
import KeyboardActionHandler from './KeyboardActionHandler'
import EntryMixin from './EntryMixin'
import EntryDragHandle from './EntryDragHandle'


export default React.createClass({
  displayName: 'ChatEntry',

  propTypes: {
    pane: React.PropTypes.instanceOf(Pane).isRequired,
    onChange: React.PropTypes.func,
  },

  mixins: [
    EntryMixin,
    Reflux.ListenerMixin,
    Reflux.connect(chat.store, 'chat'),
  ],

  getInitialState() {
    return {
      pane: this.props.pane.store.getInitialState(),
      nickText: null,
      nickFocused: false,
      empty: true,
    }
  },

  componentWillMount() {
    this.setState({empty: !this.state.chat.entryText})
  },

  componentDidMount() {
    this.listenTo(this.props.pane.store, state => this.setState({'pane': state}))
    this.listenTo(this.props.pane.focusEntry, 'focus')
    this.listenTo(this.props.pane.blurEntry, 'blur')
    const input = this.refs.input
    input.value = this.state.pane.entryText
    // in Chrome, it appears that setting the selection range can focus the
    // input without changing document.activeElement (!)
    if (this.state.pane.entrySelectionStart && this.state.pane.entrySelectionEnd) {
      input.setSelectionRange(this.state.pane.entrySelectionStart, this.state.pane.entrySelectionEnd)
    }
    this.autoSize(true)
  },

  componentDidUpdate() {
    this.autoSize()
  },

  onNickChange(ev) {
    this.setState({nickText: ev.target.value})
  },

  onNickKeyDown(ev) {
    const input = this.refs.input
    if (ev.key === 'Enter') {
      // Delay focus event to avoid double key insertion in Chrome.
      setImmediate(() => input.focus())
      ev.stopPropagation()
    } else if (ev.key === 'Escape') {
      this.setState({nickText: this.state.chat.nick}, () => input.focus())
      ev.stopPropagation()
    } else if (/^Arrow/.test(ev.key) || ev.key === 'Tab' || ev.key === 'Backspace') {
      // don't let the keyboard action handler react to these
      ev.stopPropagation()
    }
  },

  onNickFocus(ev) {
    this.setState({nickText: ev.target.value, nickFocused: true})
  },

  onNickBlur(ev) {
    actions.setNick(ev.target.value)
    this.setState({nickText: null, nickFocused: false})
  },

  onChange(ev) {
    this.saveEntryState()
    if (this.props.onChange) {
      this.props.onChange(ev)
    }
  },

  saveEntryState() {
    const input = this.refs.input
    this.props.pane.setEntryText(input.value, input.selectionStart, input.selectionEnd)
    this.setState({empty: !input.value.length})
  },

  chatSend(ev) {
    const input = this.refs.input

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

  isEmpty() {
    return this.refs.input.value.length === 0
  },

  isMultiline() {
    return /\n/.test(this.refs.input.value)
  },

  complete() {
    const input = this.refs.input
    const text = input.value
    const charRe = /\S/

    let wordEnd = input.selectionStart
    if (wordEnd < 1) {
      return
    }

    // Scan backwards for beginning of word
    let wordStart
    for (wordStart = wordEnd - 1; wordStart >= 0; wordStart--) {
      if (!charRe.test(text[wordStart])) {
        break
      }
    }
    wordStart++

    // Scan forward for the first @ sign
    let mentionStart
    for (mentionStart = wordStart; mentionStart < text.length && charRe.test(text[mentionStart]); mentionStart++) {
      if (text[mentionStart] === '@') {
        wordStart = mentionStart + 1
        break
      }
    }

    // Scan forward for end of word
    for (; wordEnd < text.length; wordEnd++) {
      if (!charRe.test(text[wordEnd])) {
        break
      }
    }

    if (wordStart >= wordEnd) {
      return
    }

    const word = text.substring(wordStart, wordEnd)
    const nameSeq = this.state.chat.who
      .toSeq()
      .filter(user => user.get('present'))
      .map(user => user.get('name', ''))
    const match = mention.rankCompletions(nameSeq, word).first()

    if (!match) {
      return
    }
    const completed = (text[wordStart - 1] !== '@' ? '@' : '') + match
    input.value = input.value.substring(0, wordStart) + completed + input.value.substring(wordEnd)
    this.saveEntryState()
  },

  autoSize(force) {
    const input = this.refs.input
    const measure = this.refs.measure
    if (force || input.value !== this.state.chat.entryText) {
      measure.style.width = input.offsetWidth + 'px'
      measure.value = input.value
      input.style.height = measure.scrollHeight + 'px'
    }
  },

  render() {
    let nick
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
        <form className={classNames('entry', 'focus-target', {'empty': this.state.empty})} onSubmit={ev => ev.preventDefault()} autoComplete="off">
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
})
