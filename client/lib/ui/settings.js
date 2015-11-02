import React from 'react'
import Reflux from 'reflux'

import storage from '../stores/storage'


export default React.createClass({
  displayName: 'Settings',

  mixins: [
    Reflux.connect(storage.store, 'storage'),
  ],

  onChangeOpenDyslexic(ev) {
    storage.set('useOpenDyslexic', ev.target.checked)
  },

  render() {
    return (
      <span className="settings-content">
        <label><input type="checkbox" checked={this.state.storage.useOpenDyslexic} onChange={this.onChangeOpenDyslexic} />dyslexia font</label>
      </span>
    )
  },
})
