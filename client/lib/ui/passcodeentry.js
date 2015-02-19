var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')


module.exports = React.createClass({
  displayName: 'PasscodeEntry',

  mixins: [
    Reflux.connect(require('../stores/chat').store, 'chat'),
    Reflux.listenTo(actions.focusEntry, 'focus'),
  ],

  focus: function() {
    this.refs.input.getDOMNode().focus()
  },

  tryPasscode: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.tryRoomPasscode(input.value)
    input.value = ''
    ev.preventDefault()
  },

  render: function() {
    return (
      <div className="entry-box passcode">
        <p className="message">This room requires a passcode.</p>
        <form className="entry" onSubmit={this.tryPasscode}>
          <label>{this.state.chat.authState == 'failed' ? 'no dice. try again:' : 'passcode:'}</label>
          <input key="passcode" ref="input" type="password" autoFocus disabled={this.state.chat.connected === false} />
        </form>
      </div>
    )
  },
})
