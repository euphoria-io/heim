var React = require('react/addons')
var Reflux = require('reflux')

var notification = require('../stores/notification')


module.exports = React.createClass({
  displayName: 'NotifyToggle',

  mixins: [
    Reflux.connect(notification.store),
  ],

  onChange: function(ev) {
    if (ev.target.checked) {
      notification.enable()
    } else {
      notification.disable()
    }
  },

  render: function() {
    return (
      <label><input type="checkbox" checked={this.state.enabled} onChange={this.onChange} />notify new messages?</label>
    )
  },
})
