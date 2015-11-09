import React from 'react'
import Immutable from 'immutable'

import FastButton from './fast-button'


export default React.createClass({
  displayName: 'AccountButton',

  propTypes: {
    account: React.PropTypes.instanceOf(Immutable.Map),
    onOpenAccountAuthDialog: React.PropTypes.func,
    onOpenAccountSettingsDialog: React.PropTypes.func,
  },

  mixins: [require('react-immutable-render-mixin')],

  render() {
    if (this.props.account) {
      return (
        <FastButton className="account-button signed-in" onClick={this.props.onOpenAccountSettingsDialog}>
          <div className="account-info">
            <div className="status">signed in</div>
            <div className="name">{this.props.account.get('name')}</div>
          </div>
        </FastButton>
      )
    }
    return <FastButton className="account-button" onClick={this.props.onOpenAccountAuthDialog}>sign in or register</FastButton>
  },
})
