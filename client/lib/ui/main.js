var React = require('react/addons')
var Reflux = require('reflux')
var cx = React.addons.classSet

var actions = require('../actions')
var Chat = require('./chat')
var NotifyToggle = require('./notifytoggle')

module.exports = React.createClass({
  mixins: [
    Reflux.connect(require('../stores/chat').store),
  ],

  send: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.sendMessage(input.value)
    input.value = ''
    ev.preventDefault()
  },

  focusInput: function() {
    this.refs.input.getDOMNode().focus()
  },

  render: function() {
    return (
      <div className="chat">
        <div className="messages-container" onClick={this.focusInput}>
          <Chat messages={this.state.messages} />
          <div className="options">
            <NotifyToggle />
          </div>
          <div className={cx({'status': true, 'disconnected': this.state.connected == false})}>disconnected!</div>
        </div>
        <form onSubmit={this.send}>
          <input ref="input" type="text" autoFocus />
        </form>
      </div>
    )
  },
})
