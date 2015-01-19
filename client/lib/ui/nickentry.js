var React = require('react')
var Reflux = require('reflux')

var actions = require('../actions')


module.exports = React.createClass({
  displayName: 'NickEntry',

  mixins: [
    require('react-immutable-render-mixin'),
    Reflux.connect(require('../stores/chat').store),
    Reflux.listenTo(actions.focusEntry, 'focus'),
  ],

  focus: function() {
    this.refs.input.getDOMNode().focus()
  },

  setNick: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.setNick(input.value)
    ev.preventDefault()
  },

  render: function() {
    return (
      <form className="entry" onSubmit={this.setNick}>
        <label>choose a nickname to start chatting:</label>
        <input key="nick" ref="input" type="text" autoFocus disabled={this.state.connected === false} />
      </form>
    )
  },
})
