import React from 'react'
import Immutable from 'immutable'
import classNames from 'classnames'

import chat from '../stores/chat'
import ui from '../stores/ui'
import MessageText from './MessageText'


export default React.createClass({
  displayName: 'UserList',

  propTypes: {
    users: React.PropTypes.instanceOf(Immutable.Map),
    selected: React.PropTypes.instanceOf(Immutable.Set),
  },

  mixins: [require('react-immutable-render-mixin')],

  onMouseDown(ev, sessionId) {
    if (ui.store.state.managerMode) {
      const selected = this.props.selected.has(sessionId)
      chat.setUserSelected(sessionId, !selected)
      ui.startToolboxSelectionDrag(!selected)
      ev.preventDefault()
    }
  },

  onMouseEnter(sessionId) {
    if (ui.store.state.managerMode && ui.store.state.draggingToolboxSelection) {
      chat.setUserSelected(sessionId, ui.store.state.draggingToolboxSelectionToggle)
    }
  },

  render() {
    let list

    list = this.props.users
      .toSeq()
      .filter(user => user.get('name'))

    list = list
      .sortBy(user => user.get('name').toLowerCase())
      .groupBy(user => /^bot:/.test(user.get('id')) ? 'bot' : 'human')

    const formatUser = user => {
      const sessionId = user.get('session_id')
      const selected = this.props.selected.has(sessionId)
      return (
        <span
          key={sessionId}
          onMouseDown={ev => this.onMouseDown(ev, sessionId)}
          onMouseEnter={() => this.onMouseEnter(sessionId)}
        >
          <MessageText
            className={classNames('nick', {'selected': selected})}
            onlyEmoji
            style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}}
            content={user.get('name')}
            title={user.get('name')}
          />
        </span>
      )
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
