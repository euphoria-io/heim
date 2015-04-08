var React = require('react/addons')
var Reflux = require('reflux')

var storage = require('../stores/storage')


module.exports = React.createClass({
  displayName: 'Settings',

  mixins: [
    Reflux.connect(storage.store, 'storage'),
  ],

  onChangeOpenDyslexic: function(ev) {
    storage.set('useOpenDyslexic', ev.target.checked)
  },

  render: function() {
    return (
      <span className="settings-content">
        <label><input type="checkbox" checked={this.state.storage.useOpenDyslexic} onChange={this.onChangeOpenDyslexic} />dyslexic font</label>
      </span>
    )
  },
})
