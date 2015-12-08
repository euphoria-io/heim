import React from 'react'
import Immutable from 'immutable'

import MessageText from './MessageText'


export default React.createClass({
  displayName: 'UserList',

  propTypes: {
    users: React.PropTypes.instanceOf(Immutable.Map),
  },

  mixins: [require('react-immutable-render-mixin')],

  render() {
    let list

    list = this.props.users
      .toSeq()
      .filter(user => user.get('name'))

    list = list
      .sortBy(user => user.get('name').toLowerCase())
      .groupBy(user => /^bot:/.test(user.get('id')) ? 'bot' : 'human')

    function formatUser(user) {
      return <MessageText
        key={user.get('session_id')}
        className="nick"
        onlyEmoji
        style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}}
        content={user.get('name')}
        title={user.get('name')}
      />
    }

    return (
      <div className="user-list" {...this.props}>
        {list.has('human') && <div className="list">
          <h1>people</h1>
          {list.get('human').map(formatUser).toIndexedSeq()}
        </div>}
        {list.has('bot') && <div className="list">
          <h1>bots</h1>
          {list.get('bot').map(formatUser).toIndexedSeq()}
        </div>}
      </div>
    )
  },
})
