var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')


module.exports = React.createClass({
  displayName: 'NickEntry',

  mixins: [
    require('react-immutable-render-mixin'),
    Reflux.connect(require('../stores/chat').store, 'chat'),
    Reflux.listenTo(actions.focusEntry, 'focus'),
  ],

  focus: function() {
    this.refs.input.getDOMNode().focus()
  },

  getInitialState: function() {
    return {nickText: ''}
  },

  setNick: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.setNick(input.value)
    setTimeout(function() {
      actions.showSettings()
    }, 250)
    ev.preventDefault()
  },

  previewNick: function() {
    var input = this.refs.input.getDOMNode()
    this.setState({nickText: input.value})
  },

  render: function() {
    // http://stackoverflow.com/a/6447935
    var isTouchDevice = 'ontouchstart' in window

    return (
      <div className="welcome">
        <div className="message">
          <h1><strong>Hello{this.state.nickText ? ' ' + this.state.nickText : ''}!</strong> <span className="no-break">Welcome to our discussion.</span></h1>
          <p>To reply to a message directly, {isTouchDevice ? 'tap' : 'use the arrow keys or click on'} it.</p>
        </div>
        <form className="entry" onSubmit={this.setNick}>
          <label>choose your name to begin:</label>
          <input key="nick" ref="input" type="text" autoFocus disabled={this.state.chat.connected === false} onChange={this.previewNick} />
        </form>
      </div>
    )
  },
})
