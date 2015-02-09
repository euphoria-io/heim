var React = require('react')
var cx = React.addons.classSet


module.exports = React.createClass({
  displayName: 'UserList',

  mixins: [require('react-immutable-render-mixin')],

  getInitialState: function() {
    return {expanded: false}
  },

  onMouseEnter: function() {
    this.setState({expanded: true})
  },

  onMouseLeave: function() {
    this.setState({expanded: false})
  },

  render: function() {
    var list
    var remaining = 0

    list = this.props.users
      .toSeq()
      .filter(user => user.get('name'))

    if (!this.state.expanded) {
      var latest = list
        .sortBy(user => -(user.get('lastSent') || 0))

      latest.cacheResult()
      remaining = latest.size - 4
      list = latest.take(4)
    }

    list = list.sortBy(user => user.get('name'))

    return (
      <div className={cx({'user-list': true, 'obscured': this.props.obscured && !this.state.expanded})} onMouseEnter={this.onMouseEnter} onMouseLeave={this.onMouseLeave}>
        {list.map(function(user) {
          return <div key={user.get('id')} className="nick" style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}}>{user.get('name')}</div>
        }, this).toArray()}
        {remaining > 0 && <div className="more nick">+{remaining} more</div>}
      </div>
    )
  },
})
