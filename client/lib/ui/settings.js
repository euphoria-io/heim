var React = require('react/addons')
var Reflux = require('reflux')

var notification = require('../stores/notification')
var storage = require('../stores/storage')


module.exports = React.createClass({
  displayName: 'Settings',

  mixins: [
    Reflux.connect(notification.store, 'notification'),
    Reflux.connect(storage.store, 'storage'),
  ],

  onChangeNotify: function(ev) {
    if (ev.target.checked) {
      notification.enablePopups()
    } else {
      notification.disablePopups()
    }
  },

  onChangeOpenDyslexic: function(ev) {
    storage.set('useOpenDyslexic', ev.target.checked)
  },

  render: function() {
    return (
      <span key="content" className="settings-content">
        <label><input type="checkbox" checked={this.state.notification.popupsEnabled} onChange={this.onChangeNotify} />notify new messages?</label>
        <label><input type="checkbox" checked={this.state.storage.useOpenDyslexic} onChange={this.onChangeOpenDyslexic} />dyslexic font</label>
      </span>
    )
  },
})
