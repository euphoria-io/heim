var React = require('react')

var MessageText = require('./message-text')


module.exports = React.createClass({
  displayName: 'UserList',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    var list

    list = this.props.users
      .toSeq()
      .filter(user => user.get('name'))

    list = list
      .sortBy(user => user.get('name').toLowerCase())
      .groupBy(user => /^bot:/.test(user.get('id')) ? 'bot' : 'human')

    function formatUser(user) {
      return <MessageText key={user.get('session_id')} className="nick" onlyEmoji={true} style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}} content={user.get('name')} title={user.get('name')} />
    }

    return (
      <div className="user-list" {...this.props}>
        {list.has('human') && <div className="list">
          <h1>people</h1>
          {list.get('human').map(formatUser).toArray()}
        </div>}
        {list.has('bot') && <div className="list">
          <h1>bots</h1>
          {list.get('bot').map(formatUser).toArray()}
        </div>}
      </div>
    )
  },
})
