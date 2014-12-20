var React = require('react')

var actions = require('../actions')


module.exports = {}

module.exports = React.createClass({
  displayName: 'ChatEntry',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./entrymixin')
  ],

  getInitialState: function() {
    return {nickText: ''}
  },

  setNick: function(ev) {
    var input = this.refs.nick.getDOMNode()
    actions.setNick(input.value)
    ev.preventDefault()
  },

  send: function(ev) {
    if (ev.which != '13') {
      return
    }

    var input = this.refs.input.getDOMNode()
    if (!input.value.length) {
      return
    }

    actions.sendMessage(input.value)
    input.value = ''
    ev.preventDefault()
  },

  previewNick: function() {
    var input = this.refs.nick.getDOMNode()
    this.setState({nickText: input.value})
  },

  render: function() {
    return (
      <form className="entry">
        <div className="nick-box">
          <div className="auto-size-container">
            <input className="nick" ref="nick" defaultValue={this.props.nick} onBlur={this.setNick} onChange={this.previewNick} />
            <span className="nick">{this.state.nickText || this.props.nick}</span>
          </div>
        </div>
        <input key="msg" ref="input" type="text" autoFocus disabled={this.state.connected === false} onKeyDown={this.send} onFocus={this.props.onFormFocus} />
      </form>
    )
  },
})
