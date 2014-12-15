var React = require('react')
var moment = require('moment')
var autolinker = require('autolinker')


module.exports = {}

module.exports = React.createClass({
  displayName: 'UserList',

  mixins: [require('react-immutable-render-mixin')],

  render: function() {
    return (
      <div className="user-list">
        {this.props.users.map(function(user, idx) {
          return <div key={user.id} className="line"><span className="nick" style={{background: 'hsl(' + user.hue + ', 65%, 85%)'}}>{user.name}</span></div>
        }, this).toArray()}
      </div>
    )
  },
})
