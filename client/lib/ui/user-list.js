var React = require('react')

var MessageText = require('./message-text')


module.exports = React.createClass({
  displayName: 'UserList',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    var list
    var remaining = 0

    list = this.props.users
      .toSeq()
      .filter(user => user.get('name'))

    if (this.props.collapsed) {
      var latest = list
        .sortBy(user => -(user.get('lastSent') || 0))

      latest.cacheResult()
      remaining = latest.size - 4
      list = latest.take(4)
    }

    list = list.sortBy(user => user.get('name').toLowerCase())

    return (
      <div className="user-list" {...this.props}>
        {list.map(function(user) {
          return <MessageText key={user.get('session_id')} className="nick" onlyEmoji={true} style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}} content={user.get('name')} />
        }, this).toArray()}
        {remaining > 0 && <div className="more nick">+{remaining} more</div>}
      </div>
    )
  },
})
