var React = require('react')

var actions = require('../actions')


module.exports = React.createClass({
  displayName: 'NickEntry',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./entrymixin')
  ],

  setNick: function(ev) {
    var input = this.refs.input.getDOMNode()
    actions.setNick(input.value)
    ev.preventDefault()
  },

  render: function() {
    return (
      <form className="entry" onSubmit={this.setNick}>
        <label>choose a nickname to start chatting:</label>
        <input key="nick" ref="input" type="text" disabled={this.props.disabled} onFocus={this.props.onFormFocus} />
      </form>
    )
  },
})
