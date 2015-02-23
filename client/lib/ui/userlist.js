var React = require('react')


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
          return <div key={user.get('id')} className="nick" style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}}>{user.get('name')}</div>
        }, this).toArray()}
        {remaining > 0 && <div className="more nick">+{remaining} more</div>}
      </div>
    )
  },
})
