var React = require('react')
var Reflux = require('reflux')


module.exports = React.createClass({
  displayName: 'KeyboardActionHandler',

  mixins: [
    Reflux.ListenerMixin,
  ],

  componentDidMount: function() {
    this.listenTo(this.props.listenTo, 'onKeyDown')
  },

  onKeyDown: function(ev) {
    var key = ev.key

    if (ev.ctrlKey) {
      key = 'Control' + key
    }

    if (ev.altKey) {
      key = 'Alt' + key
    }

    if (ev.shiftKey) {
      key = 'Shift' + key
    }

    if (ev.metaKey) {
      key = 'Meta' + key
    }

    if (key != 'Tab' && Heim.tabPressed) {
      key = 'Tab' + key
    }

    var handler = this.props.keys[key]
    if (handler && handler(ev) !== false) {
      ev.stopPropagation()
      ev.preventDefault()
    }
  },

  render: function() {
    return (
      <div onKeyDown={this.onKeyDown} {...this.props}>
        {this.props.children}
      </div>
    )
  },
})
