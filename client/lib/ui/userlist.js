var React = require('react')


module.exports = {}

module.exports = React.createClass({
  displayName: 'UserList',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    return (
      <div className="user-list">
        {this.props.users.map(function(user) {
          return <div key={user.get('id')} className="line"><span className="nick" style={{background: 'hsl(' + user.get('hue') + ', 65%, 85%)'}}>{user.get('name')}</span></div>
        }, this).toArray()}
      </div>
    )
  },
})
